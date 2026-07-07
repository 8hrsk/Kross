//go:build windows

package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/user/kross/internal/hwid"
	"golang.org/x/sys/windows/registry"
)

// windowsStore wraps fileStore and additionally writes license state
// to the Windows registry under HKCU\Software\Kross.
type windowsStore struct {
	*fileStore
}

// NewStore creates a Store backed by files in %APPDATA%\Kross
// and mirrors key data to the Windows registry.
func NewStore(appName string) (Store, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return nil, fmt.Errorf("APPDATA environment variable is not set")
	}

	baseDir := filepath.Join(appData, "Kross")
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}

	hwidHash, err := hwid.Collect()
	if err != nil {
		return nil, fmt.Errorf("collect hwid: %w", err)
	}

	fs := newFileStore(baseDir, appName, hwidHash)
	return &windowsStore{fileStore: fs}, nil
}

// SaveLicense saves to both the file system and the Windows registry.
func (ws *windowsStore) SaveLicense(data LicenseData) error {
	if err := ws.fileStore.SaveLicense(data); err != nil {
		return err
	}

	// Mirror key info to registry.
	key, _, err := registry.CreateKey(
		registry.CURRENT_USER,
		`Software\Kross`,
		registry.SET_VALUE,
	)
	if err != nil {
		return fmt.Errorf("create registry key: %w", err)
	}
	defer key.Close()

	if err := key.SetStringValue("LicenseKey", data.Key); err != nil {
		return fmt.Errorf("set registry LicenseKey: %w", err)
	}
	if err := key.SetStringValue("Email", data.Email); err != nil {
		return fmt.Errorf("set registry Email: %w", err)
	}

	return nil
}

// RemoveLicense removes from both the file system and the Windows registry.
func (ws *windowsStore) RemoveLicense() error {
	if err := ws.fileStore.RemoveLicense(); err != nil {
		return err
	}

	// Best-effort cleanup of registry values.
	_ = registry.DeleteKey(registry.CURRENT_USER, `Software\Kross`)
	return nil
}
