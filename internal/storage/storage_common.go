package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	licenseFile   = "license.dat"
	blacklistFile = "blacklist.json"
	syncQueueFile = "sync_queue.json"
	gcmNonceSize  = 12
)

// fileStore implements Store using encrypted files on disk.
type fileStore struct {
	baseDir  string
	appName  string
	hwidHash string // cached HWID hash for encryption key derivation
	mu       sync.Mutex
}

// newFileStore creates a fileStore with the given parameters.
// Exported for testing; production code should use NewStore.
func newFileStore(baseDir, appName, hwidHash string) *fileStore {
	return &fileStore{
		baseDir:  baseDir,
		appName:  appName,
		hwidHash: hwidHash,
	}
}

// deriveKey derives a 32-byte AES-256 key from the HWID hash and app name.
func (fs *fileStore) deriveKey() []byte {
	h := sha256.Sum256([]byte(fs.hwidHash + fs.appName))
	return h[:]
}

// encrypt encrypts plaintext using AES-256-GCM with a random nonce.
// Returns nonce (12 bytes) || ciphertext.
func (fs *fileStore) encrypt(plaintext []byte) ([]byte, error) {
	key := fs.deriveKey()

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cipher.NewGCM: %w", err)
	}

	nonce := make([]byte, gcmNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Format: nonce || ciphertext
	result := make([]byte, 0, len(nonce)+len(ciphertext))
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

// decrypt decrypts data produced by encrypt (nonce || ciphertext).
func (fs *fileStore) decrypt(data []byte) ([]byte, error) {
	if len(data) < gcmNonceSize {
		return nil, errors.New("ciphertext too short")
	}

	key := fs.deriveKey()

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cipher.NewGCM: %w", err)
	}

	nonce := data[:gcmNonceSize]
	ciphertext := data[gcmNonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("gcm.Open: %w", err)
	}

	return plaintext, nil
}

// filePath returns the full path to a storage file.
func (fs *fileStore) filePath(name string) string {
	return filepath.Join(fs.baseDir, name)
}

// ensureDir creates the base directory if it doesn't exist.
func (fs *fileStore) ensureDir() error {
	return os.MkdirAll(fs.baseDir, 0700)
}

// SaveLicense encrypts and persists LicenseData to disk.
func (fs *fileStore) SaveLicense(data LicenseData) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := fs.ensureDir(); err != nil {
		return fmt.Errorf("create storage dir: %w", err)
	}

	plaintext, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal license: %w", err)
	}

	encrypted, err := fs.encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("encrypt license: %w", err)
	}

	if err := os.WriteFile(fs.filePath(licenseFile), encrypted, 0600); err != nil {
		return fmt.Errorf("write license file: %w", err)
	}

	return nil
}

// LoadLicense reads and decrypts the stored license data.
// Returns ErrNoLicense if the license file does not exist.
func (fs *fileStore) LoadLicense() (LicenseData, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	var ld LicenseData

	data, err := os.ReadFile(fs.filePath(licenseFile))
	if err != nil {
		if os.IsNotExist(err) {
			return ld, ErrNoLicense
		}
		return ld, fmt.Errorf("read license file: %w", err)
	}

	plaintext, err := fs.decrypt(data)
	if err != nil {
		return ld, fmt.Errorf("decrypt license: %w", err)
	}

	if err := json.Unmarshal(plaintext, &ld); err != nil {
		return ld, fmt.Errorf("unmarshal license: %w", err)
	}

	return ld, nil
}

// RemoveLicense deletes the license file from disk.
func (fs *fileStore) RemoveLicense() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	err := os.Remove(fs.filePath(licenseFile))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove license file: %w", err)
	}
	return nil
}

// IsBlacklisted checks whether the given license ID is in the blacklist.
func (fs *fileStore) IsBlacklisted(licenseID string) (bool, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	ids, err := fs.loadBlacklist()
	if err != nil {
		return false, err
	}

	for _, id := range ids {
		if id == licenseID {
			return true, nil
		}
	}
	return false, nil
}

// AddToBlacklist appends the given license IDs to the blacklist file.
// Duplicates are silently ignored.
func (fs *fileStore) AddToBlacklist(licenseIDs []string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := fs.ensureDir(); err != nil {
		return fmt.Errorf("create storage dir: %w", err)
	}

	existing, err := fs.loadBlacklist()
	if err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(existing))
	for _, id := range existing {
		seen[id] = struct{}{}
	}

	for _, id := range licenseIDs {
		if _, ok := seen[id]; !ok {
			existing = append(existing, id)
			seen[id] = struct{}{}
		}
	}

	return fs.saveBlacklist(existing)
}

// loadBlacklist reads the blacklist file. Returns an empty slice if the file
// does not exist.
func (fs *fileStore) loadBlacklist() ([]string, error) {
	data, err := os.ReadFile(fs.filePath(blacklistFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read blacklist: %w", err)
	}

	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil, fmt.Errorf("unmarshal blacklist: %w", err)
	}
	return ids, nil
}

// saveBlacklist writes the blacklist to disk.
func (fs *fileStore) saveBlacklist(ids []string) error {
	data, err := json.Marshal(ids)
	if err != nil {
		return fmt.Errorf("marshal blacklist: %w", err)
	}
	if err := os.WriteFile(fs.filePath(blacklistFile), data, 0600); err != nil {
		return fmt.Errorf("write blacklist: %w", err)
	}
	return nil
}

// EnqueueSync adds a SyncPayload to the sync queue file.
func (fs *fileStore) EnqueueSync(payload SyncPayload) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := fs.ensureDir(); err != nil {
		return fmt.Errorf("create storage dir: %w", err)
	}

	queue, err := fs.loadSyncQueue()
	if err != nil {
		return err
	}

	queue = append(queue, payload)
	return fs.saveSyncQueue(queue)
}

// DequeueAllSync retrieves all pending sync payloads and clears the queue.
func (fs *fileStore) DequeueAllSync() ([]SyncPayload, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	queue, err := fs.loadSyncQueue()
	if err != nil {
		return nil, err
	}

	// Clear the queue file.
	if err := os.Remove(fs.filePath(syncQueueFile)); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("remove sync queue: %w", err)
	}

	return queue, nil
}

// loadSyncQueue reads the sync queue file. Returns an empty slice if the file
// does not exist.
func (fs *fileStore) loadSyncQueue() ([]SyncPayload, error) {
	data, err := os.ReadFile(fs.filePath(syncQueueFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read sync queue: %w", err)
	}

	var queue []SyncPayload
	if err := json.Unmarshal(data, &queue); err != nil {
		return nil, fmt.Errorf("unmarshal sync queue: %w", err)
	}
	return queue, nil
}

// saveSyncQueue writes the sync queue to disk.
func (fs *fileStore) saveSyncQueue(queue []SyncPayload) error {
	data, err := json.Marshal(queue)
	if err != nil {
		return fmt.Errorf("marshal sync queue: %w", err)
	}
	if err := os.WriteFile(fs.filePath(syncQueueFile), data, 0600); err != nil {
		return fmt.Errorf("write sync queue: %w", err)
	}
	return nil
}
