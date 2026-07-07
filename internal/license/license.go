// Package license provides types and functions for creating, signing,
// verifying, and encoding Kross software licenses.
package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/user/kross/internal/crypto"
)

// LicenseType represents the type of license.
type LicenseType string

const (
	// TypePersonal is a license tied to a specific email address.
	TypePersonal LicenseType = "personal"
	// TypeMass is a one-time-use license not tied to an email.
	TypeMass LicenseType = "mass"
)

// License represents a software license.
type License struct {
	ID        string      `json:"id"`         // UUID
	Type      LicenseType `json:"type"`       // "personal" or "mass"
	Email     string      `json:"email"`      // Email (empty for mass)
	IssuedAt  time.Time   `json:"issued_at"`  // Issue date
	ExpiresAt *time.Time  `json:"expires_at"` // nil = perpetual
}

// SignedLicense is a License with its Ed25519 signature.
type SignedLicense struct {
	License   License `json:"license"`
	Signature []byte  `json:"signature"`
}

// generateUUID generates a v4 UUID using crypto/rand.
func generateUUID() (string, error) {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		return "", fmt.Errorf("generate UUID: %w", err)
	}
	// Set version 4 (bits 12-15 of time_hi_and_version)
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	// Set variant RFC 4122 (bits 6-7 of clock_seq_hi_and_reserved)
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16]), nil
}

// NewPersonalLicense creates a new personal license tied to an email.
// If days <= 0, the license is perpetual (ExpiresAt = nil).
func NewPersonalLicense(email string, days int) License {
	id, err := generateUUID()
	if err != nil {
		// crypto/rand failure is unrecoverable; panic is acceptable here.
		panic(fmt.Sprintf("failed to generate UUID: %v", err))
	}

	lic := License{
		ID:       id,
		Type:     TypePersonal,
		Email:    email,
		IssuedAt: time.Now().UTC().Truncate(time.Second),
	}

	if days > 0 {
		exp := lic.IssuedAt.Add(time.Duration(days) * 24 * time.Hour)
		lic.ExpiresAt = &exp
	}

	return lic
}

// NewMassLicense creates a new mass (one-time use) license.
// Mass licenses are always perpetual and not tied to an email.
func NewMassLicense() License {
	id, err := generateUUID()
	if err != nil {
		panic(fmt.Sprintf("failed to generate UUID: %v", err))
	}

	return License{
		ID:       id,
		Type:     TypeMass,
		IssuedAt: time.Now().UTC().Truncate(time.Second),
	}
}

// SignLicense signs the license payload with the given private key.
// It serializes the License struct to JSON, signs it, and returns a SignedLicense.
func SignLicense(lic License, privateKey ed25519.PrivateKey) (SignedLicense, error) {
	payload, err := json.Marshal(lic)
	if err != nil {
		return SignedLicense{}, fmt.Errorf("marshal license for signing: %w", err)
	}

	sig := crypto.Sign(privateKey, payload)

	return SignedLicense{
		License:   lic,
		Signature: sig,
	}, nil
}

// VerifyLicense verifies the signature of a SignedLicense using the public key.
// Also checks expiration date. Returns nil if valid.
func VerifyLicense(sl SignedLicense, publicKey ed25519.PublicKey) error {
	payload, err := json.Marshal(sl.License)
	if err != nil {
		return fmt.Errorf("marshal license for verification: %w", err)
	}

	if !crypto.Verify(publicKey, payload, sl.Signature) {
		return fmt.Errorf("invalid license signature")
	}

	if sl.License.IsExpired() {
		return fmt.Errorf("license expired at %s", sl.License.ExpiresAt.Format(time.RFC3339))
	}

	return nil
}

// IsExpired checks if the license has expired.
func (l License) IsExpired() bool {
	if l.ExpiresAt == nil {
		return false
	}
	return time.Now().UTC().After(*l.ExpiresAt)
}
