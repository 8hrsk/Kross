//go:build linux

package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/user/kross/internal/hwid"
)

// NewStore creates a Store backed by files in ~/.config/.kross.
func NewStore(appName string) (Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get user home dir: %w", err)
	}

	baseDir := filepath.Join(home, ".config", ".kross")
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}

	hwidHash, err := hwid.Collect()
	if err != nil {
		return nil, fmt.Errorf("collect hwid: %w", err)
	}

	return newFileStore(baseDir, appName, hwidHash), nil
}
