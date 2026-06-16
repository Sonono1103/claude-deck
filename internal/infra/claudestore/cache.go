package claudestore

import (
	"encoding/json"
	"os"

	"github.com/Sonono1103/claude-deck/internal/infra/atomicfile"
)

const cacheVersion = 3

// record is the persisted per-file aggregate plus the file signature used to
// decide cache hits. One record corresponds to one session jsonl file.
type record struct {
	Size      int64 `json:"size"`
	MTimeMs   int64 `json:"mtime_ms"`
	LineCount int64 `json:"line_count"`

	SessionID    string `json:"session_id"`
	Title        string `json:"title"`
	CustomTitle  string `json:"custom_title"`
	Preview      string `json:"preview"`
	Model        string `json:"model"`
	CWD          string `json:"cwd"`
	GitBranch    string `json:"git_branch"`
	MessageCount int    `json:"message_count"`
	FirstTSMs    int64  `json:"first_ts_ms"`
	LastTSMs     int64  `json:"last_ts_ms"`
	IsSidechain  bool   `json:"is_sidechain"`
	Awaiting     bool   `json:"awaiting"`
}

// cache is the on-disk metadata cache, keyed by absolute jsonl path.
type cache struct {
	Version int               `json:"version"`
	Entries map[string]record `json:"entries"`
}

func loadCache(path string) cache {
	empty := cache{Version: cacheVersion, Entries: map[string]record{}}
	raw, err := os.ReadFile(path)
	if err != nil {
		return empty
	}
	var c cache
	if json.Unmarshal(raw, &c) != nil || c.Version != cacheVersion || c.Entries == nil {
		return empty
	}
	return c
}

// save writes the cache atomically.
func (c cache) save(path string) error {
	raw, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return atomicfile.Write(path, raw, 0o644)
}
