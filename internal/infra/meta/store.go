package meta

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/Sonono1103/claude-deck/internal/infra/atomicfile"
)

const version = 1

// Store holds user-applied session metadata (currently pins) in a sidecar file,
// kept separate from the read-only ~/.claude session logs.
type Store struct {
	path   string
	pinned map[string]bool
}

type fileData struct {
	Version int      `json:"version"`
	Pinned  []string `json:"pinned"`
}

func New() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	s := &Store{
		path:   filepath.Join(dir, "claude-deck", "meta.json"),
		pinned: map[string]bool{},
	}
	s.load()
	return s, nil
}

func (s *Store) load() {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var d fileData
	if json.Unmarshal(raw, &d) != nil || d.Version != version {
		return
	}
	for _, id := range d.Pinned {
		s.pinned[id] = true
	}
}

func (s *Store) IsPinned(id string) bool {
	return s.pinned[id]
}

func (s *Store) SetPinned(id string, pinned bool) error {
	if pinned {
		s.pinned[id] = true
	} else {
		delete(s.pinned, id)
	}
	return s.save()
}

func (s *Store) save() error {
	ids := slices.Sorted(maps.Keys(s.pinned))
	raw, err := json.Marshal(fileData{Version: version, Pinned: ids})
	if err != nil {
		return err
	}
	return atomicfile.Write(s.path, raw, 0o644)
}
