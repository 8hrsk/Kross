package license

import (
	"testing"
	"time"

	"github.com/user/kross/internal/crypto"
)

func TestNewPersonalLicense(t *testing.T) {
	lic := NewPersonalLicense("user@example.com", 30)

	if lic.Type != TypePersonal {
		t.Errorf("Type = %q, want %q", lic.Type, TypePersonal)
	}
	if lic.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", lic.Email, "user@example.com")
	}
	if lic.ID == "" {
		t.Error("ID is empty")
	}
	if lic.IssuedAt.IsZero() {
		t.Error("IssuedAt is zero")
	}
	if lic.ExpiresAt == nil {
		t.Fatal("ExpiresAt is nil for 30-day license")
	}

	expectedExpiry := lic.IssuedAt.Add(30 * 24 * time.Hour)
	if !lic.ExpiresAt.Equal(expectedExpiry) {
		t.Errorf("ExpiresAt = %v, want %v", lic.ExpiresAt, expectedExpiry)
	}
}

func TestNewPersonalLicense_Perpetual(t *testing.T) {
	lic := NewPersonalLicense("user@example.com", 0)

	if lic.ExpiresAt != nil {
		t.Errorf("ExpiresAt = %v, want nil for perpetual license", lic.ExpiresAt)
	}
}

func TestNewPersonalLicense_NegativeDays(t *testing.T) {
	lic := NewPersonalLicense("user@example.com", -1)

	if lic.ExpiresAt != nil {
		t.Errorf("ExpiresAt = %v, want nil for negative days", lic.ExpiresAt)
	}
}

func TestNewMassLicense(t *testing.T) {
	lic := NewMassLicense()

	if lic.Type != TypeMass {
		t.Errorf("Type = %q, want %q", lic.Type, TypeMass)
	}
	if lic.Email != "" {
		t.Errorf("Email = %q, want empty for mass license", lic.Email)
	}
	if lic.ID == "" {
		t.Error("ID is empty")
	}
	if lic.ExpiresAt != nil {
		t.Errorf("ExpiresAt = %v, want nil for mass license", lic.ExpiresAt)
	}
}

func TestSignAndVerifyLicense(t *testing.T) {
	priv, pub, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	lic := NewPersonalLicense("test@example.com", 365)
	sl, err := SignLicense(lic, priv)
	if err != nil {
		t.Fatalf("SignLicense() error: %v", err)
	}

	if len(sl.Signature) == 0 {
		t.Fatal("Signature is empty")
	}

	if err := VerifyLicense(sl, pub); err != nil {
		t.Errorf("VerifyLicense() error: %v", err)
	}
}

func TestVerifyLicense_InvalidSignature(t *testing.T) {
	_, pub, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	lic := NewPersonalLicense("test@example.com", 365)
	sl := SignedLicense{
		License:   lic,
		Signature: make([]byte, 64), // zeroed signature
	}

	if err := VerifyLicense(sl, pub); err == nil {
		t.Error("VerifyLicense() expected error for invalid signature")
	}
}

func TestVerifyLicense_TamperedLicense(t *testing.T) {
	priv, pub, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	lic := NewPersonalLicense("test@example.com", 365)
	sl, err := SignLicense(lic, priv)
	if err != nil {
		t.Fatalf("SignLicense() error: %v", err)
	}

	// Tamper with the license email.
	sl.License.Email = "hacker@example.com"

	if err := VerifyLicense(sl, pub); err == nil {
		t.Error("VerifyLicense() expected error for tampered license")
	}
}

func TestVerifyLicense_WrongKey(t *testing.T) {
	priv1, _, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}
	_, pub2, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	lic := NewMassLicense()
	sl, err := SignLicense(lic, priv1)
	if err != nil {
		t.Fatalf("SignLicense() error: %v", err)
	}

	if err := VerifyLicense(sl, pub2); err == nil {
		t.Error("VerifyLicense() expected error with wrong public key")
	}
}

func TestIsExpired_Perpetual(t *testing.T) {
	lic := NewMassLicense()
	if lic.IsExpired() {
		t.Error("IsExpired() = true for perpetual license")
	}
}

func TestIsExpired_NotYet(t *testing.T) {
	lic := NewPersonalLicense("test@example.com", 365)
	if lic.IsExpired() {
		t.Error("IsExpired() = true for license expiring in 365 days")
	}
}

func TestIsExpired_Expired(t *testing.T) {
	lic := NewPersonalLicense("test@example.com", 1)
	// Manually set expiration to the past.
	past := time.Now().UTC().Add(-24 * time.Hour)
	lic.ExpiresAt = &past

	if !lic.IsExpired() {
		t.Error("IsExpired() = false for expired license")
	}
}

func TestVerifyLicense_Expired(t *testing.T) {
	priv, pub, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	lic := NewPersonalLicense("test@example.com", 1)
	// Set expiration to the past before signing.
	past := time.Now().UTC().Add(-24 * time.Hour)
	lic.ExpiresAt = &past

	sl, err := SignLicense(lic, priv)
	if err != nil {
		t.Fatalf("SignLicense() error: %v", err)
	}

	if err := VerifyLicense(sl, pub); err == nil {
		t.Error("VerifyLicense() expected error for expired license")
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	priv, _, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	lic := NewPersonalLicense("roundtrip@example.com", 90)
	sl, err := SignLicense(lic, priv)
	if err != nil {
		t.Fatalf("SignLicense() error: %v", err)
	}

	key, err := Encode(sl)
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}

	if key[:6] != "KROSS-" {
		t.Errorf("key prefix = %q, want %q", key[:6], "KROSS-")
	}

	decoded, err := Decode(key)
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}

	if decoded.License.ID != sl.License.ID {
		t.Errorf("decoded ID = %q, want %q", decoded.License.ID, sl.License.ID)
	}
	if decoded.License.Email != sl.License.Email {
		t.Errorf("decoded Email = %q, want %q", decoded.License.Email, sl.License.Email)
	}
	if decoded.License.Type != sl.License.Type {
		t.Errorf("decoded Type = %q, want %q", decoded.License.Type, sl.License.Type)
	}
}

func TestDecode_InvalidPrefix(t *testing.T) {
	_, err := Decode("INVALID-ABCDE-FGHIJ")
	if err == nil {
		t.Error("Decode() expected error for invalid prefix")
	}
}

func TestDecode_InvalidBase32(t *testing.T) {
	_, err := Decode("KROSS-11111-!!!!!-?????")
	if err == nil {
		t.Error("Decode() expected error for invalid base32")
	}
}

func TestMassLicenseEncodeDecodeRoundTrip(t *testing.T) {
	priv, pub, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	lic := NewMassLicense()
	sl, err := SignLicense(lic, priv)
	if err != nil {
		t.Fatalf("SignLicense() error: %v", err)
	}

	key, err := Encode(sl)
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}

	decoded, err := Decode(key)
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}

	// Verify the decoded license is still valid.
	if err := VerifyLicense(decoded, pub); err != nil {
		t.Errorf("VerifyLicense(decoded) error: %v", err)
	}
}

func TestUniqueIDs(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		lic := NewMassLicense()
		if ids[lic.ID] {
			t.Fatalf("duplicate ID generated: %s", lic.ID)
		}
		ids[lic.ID] = true
	}
}
