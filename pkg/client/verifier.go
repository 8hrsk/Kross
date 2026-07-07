package client

import (
	"crypto/ed25519"
	"errors"
	"time"

	"github.com/user/kross/internal/hwid"
	"github.com/user/kross/internal/license"
	"github.com/user/kross/internal/storage"
)

var (
	ErrNoLicense     = errors.New("no active license found")
	ErrInvalidKey    = errors.New("invalid license key")
	ErrExpired       = errors.New("license has expired")
	ErrBlacklisted   = errors.New("license key has been revoked")
	ErrEmailMismatch = errors.New("email does not match the license")
)

// Config holds the configuration for the Kross client.
type Config struct {
	PublicKey ed25519.PublicKey // Embedded public key for offline verification
	ServerURL string            // Base URL of the Kross server (e.g., "https://api.example.com")
	AppName   string            // Application name (used for storage directory naming)
}

// Client is the main Kross license verification client.
type Client struct {
	config   Config
	store    storage.Store
	hwidHash string
}

// New creates a new Kross client instance.
// It initializes the platform-specific storage and collects the HWID.
func New(cfg Config) (*Client, error) {
	hwidHash, err := hwid.Collect()
	if err != nil {
		return nil, err
	}

	store, err := storage.NewStore(cfg.AppName)
	if err != nil {
		return nil, err
	}

	return &Client{
		config:   cfg,
		store:    store,
		hwidHash: hwidHash,
	}, nil
}

// CheckLicense checks if a valid, non-blacklisted license exists in storage.
// Returns nil if the license is valid.
func (c *Client) CheckLicense() error {
	data, err := c.store.LoadLicense()
	if err != nil {
		if errors.Is(err, storage.ErrNoLicense) {
			return ErrNoLicense
		}
		return err
	}

	signedLic, err := license.Decode(data.Key)
	if err != nil {
		return ErrInvalidKey
	}

	if signedLic.License.IsExpired() {
		return ErrExpired
	}

	if err := license.VerifyLicense(signedLic, c.config.PublicKey); err != nil {
		// If VerifyLicense returns an error, it could be either signature or expiration.
		// Since we already checked expiration, it must be signature (or expired, which we'll handle gracefully).
		if signedLic.License.IsExpired() {
			return ErrExpired
		}
		return ErrInvalidKey
	}

	isBlacklisted, err := c.store.IsBlacklisted(signedLic.License.ID)
	if err != nil {
		return err
	}
	if isBlacklisted {
		return ErrBlacklisted
	}

	return nil
}

// Activate validates the provided key and email, saves the license if valid,
// and enqueues a sync payload for background server communication.
func (c *Client) Activate(keyString string, email string) error {
	signedLic, err := license.Decode(keyString)
	if err != nil {
		return ErrInvalidKey
	}

	if signedLic.License.IsExpired() {
		return ErrExpired
	}

	if err := license.VerifyLicense(signedLic, c.config.PublicKey); err != nil {
		if signedLic.License.IsExpired() {
			return ErrExpired
		}
		return ErrInvalidKey
	}

	if signedLic.License.Type == license.TypePersonal && signedLic.License.Email != "" {
		if signedLic.License.Email != email {
			return ErrEmailMismatch
		}
	}

	isBlacklisted, err := c.store.IsBlacklisted(signedLic.License.ID)
	if err != nil {
		return err
	}
	if isBlacklisted {
		return ErrBlacklisted
	}

	err = c.store.SaveLicense(storage.LicenseData{
		Key:         keyString,
		Email:       email,
		ActivatedAt: time.Now(),
	})
	if err != nil {
		return err
	}

	err = c.store.EnqueueSync(storage.SyncPayload{
		LicenseID:   signedLic.License.ID,
		LicenseType: string(signedLic.License.Type),
		Email:       email,
		HWIDHash:    c.hwidHash,
		Timestamp:   time.Now(),
	})
	if err != nil {
		return err
	}

	return nil
}

// GetHWID returns the machine's HWID hash.
func (c *Client) GetHWID() string {
	return c.hwidHash
}
