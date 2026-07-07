package hwid

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Collect returns a SHA-256 hash of the machine's hardware identity.
// The hash is hex-encoded (64 characters).
func Collect() (string, error) {
	raw, err := collectRaw()
	if err != nil {
		return "", fmt.Errorf("hwid: failed to collect hardware ID: %w", err)
	}
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:]), nil
}
