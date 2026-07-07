package client

import (
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/user/kross/internal/crypto"
	"github.com/user/kross/internal/license"
	"github.com/user/kross/internal/storage"
)

type mockStore struct {
	license   *storage.LicenseData
	blacklist map[string]bool
	syncQueue []storage.SyncPayload
}

func (m *mockStore) SaveLicense(data storage.LicenseData) error {
	m.license = &data
	return nil
}

func (m *mockStore) LoadLicense() (storage.LicenseData, error) {
	if m.license == nil {
		return storage.LicenseData{}, storage.ErrNoLicense
	}
	return *m.license, nil
}

func (m *mockStore) RemoveLicense() error {
	m.license = nil
	return nil
}

func (m *mockStore) IsBlacklisted(licenseID string) (bool, error) {
	return m.blacklist[licenseID], nil
}

func (m *mockStore) AddToBlacklist(licenseIDs []string) error {
	for _, id := range licenseIDs {
		m.blacklist[id] = true
	}
	return nil
}

func (m *mockStore) EnqueueSync(payload storage.SyncPayload) error {
	m.syncQueue = append(m.syncQueue, payload)
	return nil
}

func (m *mockStore) DequeueAllSync() ([]storage.SyncPayload, error) {
	q := m.syncQueue
	m.syncQueue = nil
	return q, nil
}

func setupTestClient(t *testing.T) (*Client, ed25519.PrivateKey) {
	priv, pub, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	c := &Client{
		config: Config{
			PublicKey: pub,
			ServerURL: "http://localhost:8080",
			AppName:   "TestApp",
		},
		store: &mockStore{
			blacklist: make(map[string]bool),
		},
		hwidHash: "dummy-hwid",
	}

	return c, priv
}

func TestCheckLicense_NoLicense(t *testing.T) {
	c, _ := setupTestClient(t)
	err := c.CheckLicense()
	if err != ErrNoLicense {
		t.Errorf("Expected ErrNoLicense, got: %v", err)
	}
}

func TestActivate_ValidPersonalLicense(t *testing.T) {
	c, priv := setupTestClient(t)
	
	lic := license.NewPersonalLicense("test@example.com", 30)
	signedLic, _ := license.SignLicense(lic, priv)
	encoded, _ := license.Encode(signedLic)

	err := c.Activate(encoded, "test@example.com")
	if err != nil {
		t.Errorf("Activate failed: %v", err)
	}

	err = c.CheckLicense()
	if err != nil {
		t.Errorf("CheckLicense failed after activation: %v", err)
	}
}

func TestActivate_InvalidKey(t *testing.T) {
	c, _ := setupTestClient(t)
	err := c.Activate("INVALID-KEY", "test@example.com")
	if err != ErrInvalidKey {
		t.Errorf("Expected ErrInvalidKey, got: %v", err)
	}
}

func TestActivate_EmailMismatch(t *testing.T) {
	c, priv := setupTestClient(t)
	
	lic := license.NewPersonalLicense("test@example.com", 30)
	signedLic, _ := license.SignLicense(lic, priv)
	encoded, _ := license.Encode(signedLic)

	err := c.Activate(encoded, "wrong@example.com")
	if err != ErrEmailMismatch {
		t.Errorf("Expected ErrEmailMismatch, got: %v", err)
	}
}

func TestCheckLicense_Expired(t *testing.T) {
	c, priv := setupTestClient(t)
	
	// Create an expired license
	lic := license.NewPersonalLicense("test@example.com", 0)
	past := time.Now().Add(-24 * time.Hour)
	lic.ExpiresAt = &past
	
	signedLic, _ := license.SignLicense(lic, priv)
	encoded, _ := license.Encode(signedLic)

	// Bypass Activate since it also checks expiration during verify
	c.store.SaveLicense(storage.LicenseData{
		Key:   encoded,
		Email: "test@example.com",
	})

	err := c.CheckLicense()
	if err != ErrExpired {
		t.Errorf("Expected ErrExpired, got: %v", err)
	}
}

func TestCheckLicense_Blacklisted(t *testing.T) {
	c, priv := setupTestClient(t)
	
	lic := license.NewPersonalLicense("test@example.com", 30)
	signedLic, _ := license.SignLicense(lic, priv)
	encoded, _ := license.Encode(signedLic)

	c.Activate(encoded, "test@example.com")
	c.store.AddToBlacklist([]string{lic.ID})

	err := c.CheckLicense()
	if err != ErrBlacklisted {
		t.Errorf("Expected ErrBlacklisted, got: %v", err)
	}
}
