package hwid

import (
	"encoding/hex"
	"testing"
)

func TestCollect_ReturnsValidHexHash(t *testing.T) {
	hash, err := Collect()
	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	if len(hash) != 64 {
		t.Errorf("expected 64-character hex string, got %d characters: %q", len(hash), hash)
	}

	if _, err := hex.DecodeString(hash); err != nil {
		t.Errorf("hash is not valid hex: %v", err)
	}
}

func TestCollect_IsDeterministic(t *testing.T) {
	hash1, err := Collect()
	if err != nil {
		t.Fatalf("first Collect() returned error: %v", err)
	}

	hash2, err := Collect()
	if err != nil {
		t.Fatalf("second Collect() returned error: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Collect() is not deterministic: %q != %q", hash1, hash2)
	}
}
