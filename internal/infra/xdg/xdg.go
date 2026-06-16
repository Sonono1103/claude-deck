// Package xdg resolves XDG base directories shared across claude-deck's infra.
package xdg

import (
	"os"
	"path/filepath"
)

// DataDir returns the XDG data home ($XDG_DATA_HOME, else ~/.local/share).
func DataDir() (string, error) {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}
