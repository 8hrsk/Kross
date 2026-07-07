// Package crypto provides Ed25519 key generation, signing, and verification
// utilities for the Kross license system.
package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
)

// GenerateKeyPair generates a new Ed25519 key pair.
// Returns the private key and public key as byte slices.
func GenerateKeyPair() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate ed25519 key pair: %w", err)
	}
	return priv, pub, nil
}

// SavePrivateKey saves the private key to a PEM-encoded file.
func SavePrivateKey(path string, key ed25519.PrivateKey) error {
	block := &pem.Block{
		Type:  "ED25519 PRIVATE KEY",
		Bytes: key,
	}
	data := pem.EncodeToMemory(block)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("save private key: %w", err)
	}
	return nil
}

// SavePublicKey saves the public key to a PEM-encoded file.
func SavePublicKey(path string, key ed25519.PublicKey) error {
	block := &pem.Block{
		Type:  "ED25519 PUBLIC KEY",
		Bytes: key,
	}
	data := pem.EncodeToMemory(block)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("save public key: %w", err)
	}
	return nil
}

// LoadPrivateKey loads a PEM-encoded private key from a file.
func LoadPrivateKey(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decode private key PEM: no PEM block found")
	}
	if block.Type != "ED25519 PRIVATE KEY" {
		return nil, fmt.Errorf("decode private key PEM: unexpected block type %q", block.Type)
	}
	if len(block.Bytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("decode private key PEM: invalid key size %d", len(block.Bytes))
	}

	return ed25519.PrivateKey(block.Bytes), nil
}

// LoadPublicKey loads a PEM-encoded public key from a file.
func LoadPublicKey(path string) (ed25519.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decode public key PEM: no PEM block found")
	}
	if block.Type != "ED25519 PUBLIC KEY" {
		return nil, fmt.Errorf("decode public key PEM: unexpected block type %q", block.Type)
	}
	if len(block.Bytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("decode public key PEM: invalid key size %d", len(block.Bytes))
	}

	return ed25519.PublicKey(block.Bytes), nil
}

// Sign signs the given data with the private key.
func Sign(privateKey ed25519.PrivateKey, data []byte) []byte {
	return ed25519.Sign(privateKey, data)
}

// Verify verifies the signature of the given data with the public key.
func Verify(publicKey ed25519.PublicKey, data []byte, signature []byte) bool {
	return ed25519.Verify(publicKey, data, signature)
}
