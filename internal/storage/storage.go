package storage

import (
	"errors"
	"time"
)

// ErrNoLicense is returned when no license data is found on disk.
var ErrNoLicense = errors.New("no license found")

// SyncPayload represents data to be sent to the server.
type SyncPayload struct {
	LicenseID   string    `json:"license_id"`
	LicenseType string    `json:"license_type"`
	Email       string    `json:"email"`
	HWIDHash    string    `json:"hwid_hash"`
	Timestamp   time.Time `json:"timestamp"`
}

// LicenseData represents stored license activation data.
type LicenseData struct {
	Key         string    `json:"key"`
	Email       string    `json:"email"`
	ActivatedAt time.Time `json:"activated_at"`
}

// Store defines the interface for platform-specific license storage.
type Store interface {
	// SaveLicense persists the given license data to disk (encrypted).
	SaveLicense(data LicenseData) error

	// LoadLicense retrieves the stored license data.
	// Returns ErrNoLicense if no license is saved.
	LoadLicense() (LicenseData, error)

	// RemoveLicense deletes the stored license data.
	RemoveLicense() error

	// IsBlacklisted checks whether the given license ID is in the blacklist.
	IsBlacklisted(licenseID string) (bool, error)

	// AddToBlacklist appends the given license IDs to the blacklist.
	AddToBlacklist(licenseIDs []string) error

	// EnqueueSync adds a sync payload to the pending queue.
	EnqueueSync(payload SyncPayload) error

	// DequeueAllSync retrieves and removes all pending sync payloads.
	DequeueAllSync() ([]SyncPayload, error)
}
