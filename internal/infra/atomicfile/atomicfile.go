// Package atomicfile writes files atomically via a temp file + rename.
package atomicfile

import (
	"os"
	"path/filepath"
)

// Write creates parent directories and writes data to path atomically: it writes
// to a temp file in the same directory, then renames it over path.
func Write(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
