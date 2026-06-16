package claudestore

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Sonono1103/claude-deck/internal/domain"
)

const maxLineBytes = 64 * 1024 * 1024

// Repository reads Claude Code sessions from ~/.claude/projects, backed by an
// append-aware metadata cache.
type Repository struct {
	root      string
	cachePath string
	cache     cache
	dirty     bool
}

// New builds a Repository pointed at the user's ~/.claude/projects directory.
func New() (*Repository, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	cachePath := filepath.Join(cacheDir, "claude-deck", fmt.Sprintf("cache-v%d.json", cacheVersion))
	return &Repository{
		root:      filepath.Join(home, ".claude", "projects"),
		cachePath: cachePath,
		cache:     loadCache(cachePath),
	}, nil
}

// List returns all sessions, using the cache where files are unchanged and
// parsing only appended bytes where files have grown.
func (r *Repository) List() ([]domain.Session, error) {
	files, err := filepath.Glob(filepath.Join(r.root, "*", "*.jsonl"))
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(files))
	out := make([]domain.Session, 0, len(files))

	for _, path := range files {
		seen[path] = true
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		size := info.Size()
		mtimeMs := info.ModTime().UnixMilli()

		rec, cached := r.cache.Entries[path]
		switch {
		case cached && rec.Size == size && rec.MTimeMs == mtimeMs:
			// full hit: nothing to do
		case cached && size > rec.Size && mtimeMs >= rec.MTimeMs:
			// append-only growth: parse just the new tail
			if n, err := parseInto(&rec, path, rec.Size); err == nil {
				rec.LineCount += n
			} else {
				rec = fullParse(path)
			}
			rec.Size, rec.MTimeMs = size, mtimeMs
			r.cache.Entries[path] = rec
			r.dirty = true
		default:
			rec = fullParse(path)
			rec.Size, rec.MTimeMs = size, mtimeMs
			r.cache.Entries[path] = rec
			r.dirty = true
		}

		out = append(out, recordToSession(path, rec))
	}

	for path := range r.cache.Entries {
		if !seen[path] {
			delete(r.cache.Entries, path)
			r.dirty = true
		}
	}

	if r.dirty {
		_ = r.cache.save(r.cachePath)
		r.dirty = false
	}
	return out, nil
}

func fullParse(path string) record {
	var rec record
	n, _ := parseInto(&rec, path, 0)
	rec.LineCount = n
	return rec
}

// rawLine mirrors the subset of jsonl fields we aggregate.
type rawLine struct {
	Type        string          `json:"type"`
	SessionID   string          `json:"sessionId"`
	AITitle     string          `json:"aiTitle"`
	CustomTitle string          `json:"customTitle"`
	LastPrompt  string          `json:"lastPrompt"`
	CWD         string          `json:"cwd"`
	GitBranch   string          `json:"gitBranch"`
	Timestamp   string          `json:"timestamp"`
	IsSidechain bool            `json:"isSidechain"`
	Message     json.RawMessage `json:"message"`
}

// parseInto folds lines from startOffset onward into rec, returning the number
// of lines parsed. startOffset must be a line boundary (previous file size).
func parseInto(rec *record, path string, startOffset int64) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	if startOffset > 0 {
		if _, err := f.Seek(startOffset, io.SeekStart); err != nil {
			return 0, err
		}
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), maxLineBytes)

	var parsed int64
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var l rawLine
		if json.Unmarshal(line, &l) != nil {
			continue
		}
		foldLine(rec, &l)
		parsed++
	}
	return parsed, sc.Err()
}

func foldLine(rec *record, l *rawLine) {
	switch l.Type {
	case "ai-title":
		if l.AITitle != "" {
			rec.Title = l.AITitle
		}
	case "custom-title":
		// A user-supplied rename. Always wins over the AI title, even though
		// Claude Code writes a fresh ai-title line right after each rename.
		rec.CustomTitle = l.CustomTitle
	case "last-prompt":
		if l.LastPrompt != "" {
			rec.Preview = l.LastPrompt
		}
	case "user", "assistant":
		rec.MessageCount++
		if rec.SessionID == "" {
			rec.SessionID = l.SessionID
		}
		if rec.CWD == "" {
			rec.CWD = l.CWD
		}
		if rec.GitBranch == "" && l.GitBranch != "" {
			rec.GitBranch = l.GitBranch
		}
		if l.IsSidechain {
			rec.IsSidechain = true
		}
		if ts := parseTSMs(l.Timestamp); ts > 0 {
			if rec.FirstTSMs == 0 || ts < rec.FirstTSMs {
				rec.FirstTSMs = ts
			}
			if ts > rec.LastTSMs {
				rec.LastTSMs = ts
			}
		}
		switch l.Type {
		case "user":
			// A user/tool-result turn means the ball is back with Claude.
			rec.Awaiting = false
		case "assistant":
			if len(l.Message) > 0 {
				var m struct {
					Model      string `json:"model"`
					StopReason string `json:"stop_reason"`
				}
				// Skip "<synthetic>" (and model-less) lines — Claude Code's
				// marker for assistant turns it fabricated locally (interrupts,
				// "no response requested") rather than received from a model.
				// They must not clobber Model or the end_turn waiting signal.
				if json.Unmarshal(l.Message, &m) == nil && m.Model != "" && m.Model != "<synthetic>" {
					rec.Model = m.Model
					// end_turn = Claude finished its turn and is now waiting
					// for the user to respond.
					rec.Awaiting = m.StopReason == "end_turn"
				}
			}
		}
	}
}

func parseTSMs(ts string) int64 {
	if ts == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}

func recordToSession(path string, rec record) domain.Session {
	id := rec.SessionID
	if id == "" {
		id = sessionIDFromPath(path)
	}
	s := domain.Session{
		ID:           id,
		Source:       domain.SourceClaude,
		FilePath:     path,
		Title:        rec.Title,
		Preview:      rec.Preview,
		Model:        rec.Model,
		CWD:          rec.CWD,
		GitBranch:    rec.GitBranch,
		MessageCount: rec.MessageCount,
		IsSidechain:  rec.IsSidechain,
		Awaiting:     rec.Awaiting,
	}
	if rec.FirstTSMs > 0 {
		s.FirstTS = time.UnixMilli(rec.FirstTSMs)
	}
	if rec.LastTSMs > 0 {
		s.LastTS = time.UnixMilli(rec.LastTSMs)
	}
	// Title precedence: user rename > AI title > last prompt.
	if rec.CustomTitle != "" {
		s.Title = rec.CustomTitle
	}
	if s.Title == "" {
		s.Title = rec.Preview
	}
	return s
}

func sessionIDFromPath(path string) string {
	base := filepath.Base(path)
	return base[:len(base)-len(filepath.Ext(base))]
}
