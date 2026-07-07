package crypto

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}
	if len(priv) != ed25519.PrivateKeySize {
		t.Errorf("private key size = %d, want %d", len(priv), ed25519.PrivateKeySize)
	}
	if len(pub) != ed25519.PublicKeySize {
		t.Errorf("public key size = %d, want %d", len(pub), ed25519.PublicKeySize)
	}
}

func TestSaveLoadPrivateKey(t *testing.T) {
	priv, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "private.pem")

	if err := SavePrivateKey(path, priv); err != nil {
		t.Fatalf("SavePrivateKey() error: %v", err)
	}

	loaded, err := LoadPrivateKey(path)
	if err != nil {
		t.Fatalf("LoadPrivateKey() error: %v", err)
	}

	if !priv.Equal(loaded) {
		t.Error("loaded private key does not match original")
	}

	// Verify file permissions (owner read/write only).
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("private key file permissions = %o, want 0600", perm)
	}
}

func TestSaveLoadPublicKey(t *testing.T) {
	_, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "public.pem")

	if err := SavePublicKey(path, pub); err != nil {
		t.Fatalf("SavePublicKey() error: %v", err)
	}

	loaded, err := LoadPublicKey(path)
	if err != nil {
		t.Fatalf("LoadPublicKey() error: %v", err)
	}

	if !pub.Equal(loaded) {
		t.Error("loaded public key does not match original")
	}
}

func TestLoadPrivateKey_NotFound(t *testing.T) {
	_, err := LoadPrivateKey("/nonexistent/path.pem")
	if err == nil {
		t.Fatal("LoadPrivateKey() expected error for nonexistent file")
	}
}

func TestLoadPublicKey_InvalidPEM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.pem")
	if err := os.WriteFile(path, []byte("not a pem file"), 0644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	_, err := LoadPublicKey(path)
	if err == nil {
		t.Fatal("LoadPublicKey() expected error for invalid PEM")
	}
}

func TestSignAndVerify(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	data := []byte("license payload data")
	sig := Sign(priv, data)

	if !Verify(pub, data, sig) {
		t.Error("Verify() returned false for valid signature")
	}
}

func TestVerify_InvalidSignature(t *testing.T) {
	_, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	data := []byte("license payload data")
	badSig := make([]byte, ed25519.SignatureSize)

	if Verify(pub, data, badSig) {
		t.Error("Verify() returned true for invalid signature")
	}
}

func TestVerify_TamperedData(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	data := []byte("original data")
	sig := Sign(priv, data)

	tampered := []byte("tampered data")
	if Verify(pub, tampered, sig) {
		t.Error("Verify() returned true for tampered data")
	}
}

func TestVerify_WrongKey(t *testing.T) {
	priv1, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}
	_, pub2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	data := []byte("test data")
	sig := Sign(priv1, data)

	if Verify(pub2, data, sig) {
		t.Error("Verify() returned true with wrong public key")
	}
}
