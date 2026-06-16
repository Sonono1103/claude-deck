package trash

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Sonono1103/claude-deck/internal/domain"
	"github.com/Sonono1103/claude-deck/internal/infra/xdg"
)

// Trash moves deleted session files into a trash directory instead of removing
// them outright, so a deletion can be undone later.
type Trash struct {
	dir string
}

type meta struct {
	OriginalPath string `json:"original_path"`
	SessionID    string `json:"session_id"`
	Title        string `json:"title"`
	DeletedAt    string `json:"deleted_at"`
}

func New() (*Trash, error) {
	base, err := xdg.DataDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(base, "claude-deck", "trash")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Trash{dir: dir}, nil
}

// Trash moves the session's file into the trash dir and records its origin.
func (t *Trash) Trash(s domain.Session) error {
	dest := filepath.Join(t.dir, s.ID+".jsonl")
	if err := move(s.FilePath, dest); err != nil {
		return err
	}

	m := meta{
		OriginalPath: s.FilePath,
		SessionID:    s.ID,
		Title:        s.Title,
		DeletedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil // file already moved; metadata is best-effort
	}
	_ = os.WriteFile(filepath.Join(t.dir, s.ID+".meta.json"), raw, 0o644)
	return nil
}

// move renames across paths, falling back to copy+remove for cross-device moves.
func move(src, dest string) error {
	if err := os.Rename(src, dest); err == nil {
		return nil
	}
	if err := copyFile(src, dest); err != nil {
		return err
	}
	return os.Remove(src)
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
