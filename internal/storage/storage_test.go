package storage

import (
	"errors"
	"testing"
	"time"
)

const (
	testAppName  = "kross-test"
	testHWIDHash = "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
)

func newTestStore(t *testing.T) *fileStore {
	t.Helper()
	dir := t.TempDir()
	return newFileStore(dir, testAppName, testHWIDHash)
}

func TestNewStore_CreatesBaseDirectory(t *testing.T) {
	store, err := NewStore("kross-integration-test")
	if err != nil {
		t.Fatalf("NewStore() returned error: %v", err)
	}

	// Verify the store is usable by doing a simple operation.
	_, err = store.LoadLicense()
	if !errors.Is(err, ErrNoLicense) {
		t.Errorf("expected ErrNoLicense from fresh store, got: %v", err)
	}
}

func TestLoadLicense_ReturnsErrNoLicense_WhenEmpty(t *testing.T) {
	store := newTestStore(t)

	_, err := store.LoadLicense()
	if !errors.Is(err, ErrNoLicense) {
		t.Errorf("expected ErrNoLicense, got: %v", err)
	}
}

func TestSaveLicense_LoadLicense_RoundTrip(t *testing.T) {
	store := newTestStore(t)

	want := LicenseData{
		Key:         "TEST-LICENSE-KEY-1234",
		Email:       "user@example.com",
		ActivatedAt: time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC),
	}

	if err := store.SaveLicense(want); err != nil {
		t.Fatalf("SaveLicense() returned error: %v", err)
	}

	got, err := store.LoadLicense()
	if err != nil {
		t.Fatalf("LoadLicense() returned error: %v", err)
	}

	if got.Key != want.Key {
		t.Errorf("Key: got %q, want %q", got.Key, want.Key)
	}
	if got.Email != want.Email {
		t.Errorf("Email: got %q, want %q", got.Email, want.Email)
	}
	if !got.ActivatedAt.Equal(want.ActivatedAt) {
		t.Errorf("ActivatedAt: got %v, want %v", got.ActivatedAt, want.ActivatedAt)
	}
}

func TestSaveLicense_EncryptsData(t *testing.T) {
	store := newTestStore(t)

	data := LicenseData{
		Key:         "ENCRYPTED-KEY-5678",
		Email:       "encrypted@example.com",
		ActivatedAt: time.Now().UTC(),
	}

	if err := store.SaveLicense(data); err != nil {
		t.Fatalf("SaveLicense() returned error: %v", err)
	}

	// A different HWID hash should fail to decrypt.
	store2 := newFileStore(store.baseDir, testAppName, "different-hwid-hash-value")
	_, err := store2.LoadLicense()
	if err == nil {
		t.Error("expected decryption to fail with wrong HWID hash, but it succeeded")
	}
}

func TestRemoveLicense(t *testing.T) {
	store := newTestStore(t)

	data := LicenseData{
		Key:         "TO-BE-REMOVED",
		Email:       "remove@example.com",
		ActivatedAt: time.Now().UTC(),
	}

	if err := store.SaveLicense(data); err != nil {
		t.Fatalf("SaveLicense() returned error: %v", err)
	}

	if err := store.RemoveLicense(); err != nil {
		t.Fatalf("RemoveLicense() returned error: %v", err)
	}

	_, err := store.LoadLicense()
	if !errors.Is(err, ErrNoLicense) {
		t.Errorf("expected ErrNoLicense after removal, got: %v", err)
	}
}

func TestBlacklist_AddAndCheck(t *testing.T) {
	store := newTestStore(t)

	// Initially nothing is blacklisted.
	blacklisted, err := store.IsBlacklisted("license-a")
	if err != nil {
		t.Fatalf("IsBlacklisted() returned error: %v", err)
	}
	if blacklisted {
		t.Error("expected license-a to not be blacklisted initially")
	}

	// Add some IDs.
	if err := store.AddToBlacklist([]string{"license-a", "license-b"}); err != nil {
		t.Fatalf("AddToBlacklist() returned error: %v", err)
	}

	// Check blacklisted.
	blacklisted, err = store.IsBlacklisted("license-a")
	if err != nil {
		t.Fatalf("IsBlacklisted() returned error: %v", err)
	}
	if !blacklisted {
		t.Error("expected license-a to be blacklisted")
	}

	blacklisted, err = store.IsBlacklisted("license-b")
	if err != nil {
		t.Fatalf("IsBlacklisted() returned error: %v", err)
	}
	if !blacklisted {
		t.Error("expected license-b to be blacklisted")
	}

	// Check non-blacklisted.
	blacklisted, err = store.IsBlacklisted("license-c")
	if err != nil {
		t.Fatalf("IsBlacklisted() returned error: %v", err)
	}
	if blacklisted {
		t.Error("expected license-c to NOT be blacklisted")
	}
}

func TestBlacklist_NoDuplicates(t *testing.T) {
	store := newTestStore(t)

	if err := store.AddToBlacklist([]string{"dup-id", "dup-id"}); err != nil {
		t.Fatalf("AddToBlacklist() returned error: %v", err)
	}
	if err := store.AddToBlacklist([]string{"dup-id"}); err != nil {
		t.Fatalf("second AddToBlacklist() returned error: %v", err)
	}

	// Verify via loading raw data that there's only one entry.
	ids, err := store.loadBlacklist()
	if err != nil {
		t.Fatalf("loadBlacklist() returned error: %v", err)
	}
	if len(ids) != 1 {
		t.Errorf("expected 1 blacklist entry, got %d: %v", len(ids), ids)
	}
}

func TestSyncQueue_EnqueueAndDequeue(t *testing.T) {
	store := newTestStore(t)

	now := time.Now().UTC().Truncate(time.Second)

	p1 := SyncPayload{
		LicenseID:   "lic-1",
		LicenseType: "pro",
		Email:       "user1@example.com",
		HWIDHash:    "hash1",
		Timestamp:   now,
	}
	p2 := SyncPayload{
		LicenseID:   "lic-2",
		LicenseType: "trial",
		Email:       "user2@example.com",
		HWIDHash:    "hash2",
		Timestamp:   now.Add(time.Hour),
	}

	if err := store.EnqueueSync(p1); err != nil {
		t.Fatalf("EnqueueSync(p1) returned error: %v", err)
	}
	if err := store.EnqueueSync(p2); err != nil {
		t.Fatalf("EnqueueSync(p2) returned error: %v", err)
	}

	payloads, err := store.DequeueAllSync()
	if err != nil {
		t.Fatalf("DequeueAllSync() returned error: %v", err)
	}

	if len(payloads) != 2 {
		t.Fatalf("expected 2 payloads, got %d", len(payloads))
	}

	if payloads[0].LicenseID != "lic-1" {
		t.Errorf("first payload LicenseID: got %q, want %q", payloads[0].LicenseID, "lic-1")
	}
	if payloads[1].LicenseID != "lic-2" {
		t.Errorf("second payload LicenseID: got %q, want %q", payloads[1].LicenseID, "lic-2")
	}

	// Queue should be empty after dequeue.
	payloads, err = store.DequeueAllSync()
	if err != nil {
		t.Fatalf("second DequeueAllSync() returned error: %v", err)
	}
	if len(payloads) != 0 {
		t.Errorf("expected empty queue after dequeue, got %d payloads", len(payloads))
	}
}
