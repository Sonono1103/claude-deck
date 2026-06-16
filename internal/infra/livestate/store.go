// Package livestate stores live per-session lifecycle state reported by Claude
// Code hooks, so the TUI can tell which sessions are currently waiting for input.
package livestate

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"github.com/Sonono1103/claude-deck/internal/domain"
	"github.com/Sonono1103/claude-deck/internal/infra/atomicfile"
	"github.com/Sonono1103/claude-deck/internal/infra/xdg"
)

const version = 1

// Entry is the live state for one session.
type Entry struct {
	State     domain.LiveState `json:"state"`
	CWD       string           `json:"cwd"`
	PID       int              `json:"pid"` // the claude process (a hook's parent), for liveness
	UpdatedAt int64            `json:"updated_at"`
}

type file struct {
	Version  int              `json:"version"`
	Sessions map[string]Entry `json:"sessions"`
}

func storePath() (string, error) {
	dir, err := xdg.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "claude-deck", "live.json"), nil
}

// Load returns the current session states (best-effort; no lock).
func Load() (map[string]Entry, error) {
	p, err := storePath()
	if err != nil {
		return nil, err
	}
	return loadFile(p), nil
}

func loadFile(path string) map[string]Entry {
	m, _ := loadFileStrict(path)
	return m
}

// loadFileStrict is loadFile that surfaces a transient read error instead of
// swallowing it as an empty map. A missing file is a legitimate empty store; a
// read error is not, and must not be mistaken for "no sessions" under the
// Upsert lock — otherwise the rewrite would wipe every other live session.
// Corrupt or stale-version contents are unrecoverable, so they reset to empty.
func loadFileStrict(path string) (map[string]Entry, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	var f file
	if json.Unmarshal(raw, &f) != nil || f.Version != version || f.Sessions == nil {
		return map[string]Entry{}, nil
	}
	return f.Sessions, nil
}

// Upsert mutates the session map under an exclusive lock and saves it. mutate
// reports whether it changed anything; when it returns false the file is left
// untouched, so repeated no-op events (e.g. a PreToolUse stream on an already
// running session) don't rewrite the store.
func Upsert(mutate func(map[string]Entry) bool) error {
	p, err := storePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return withLock(p+".lock", func() error {
		m, err := loadFileStrict(p)
		if err != nil {
			return err // don't clobber other sessions on a transient read error
		}
		if !mutate(m) {
			return nil
		}
		raw, err := json.Marshal(file{Version: version, Sessions: m})
		if err != nil {
			return err
		}
		return atomicfile.Write(p, raw, 0o644)
	})
}

// Alive reports whether a process is still running.
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil || err == syscall.EPERM
}

func withLock(lockPath string, fn func() error) error {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}
