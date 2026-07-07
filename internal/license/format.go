package license

import (
	"encoding/base32"
	"encoding/json"
	"fmt"
	"strings"
)

// encoder is Base32 StdEncoding without padding.
var encoder = base32.StdEncoding.WithPadding(base32.NoPadding)

const keyPrefix = "KROSS-"

// Encode encodes a SignedLicense into a user-friendly key string.
// Format: KROSS-XXXXX-XXXXX-XXXXX-... (Base32 encoded, grouped by 5 chars, separated by dashes).
func Encode(sl SignedLicense) (string, error) {
	data, err := json.Marshal(sl)
	if err != nil {
		return "", fmt.Errorf("encode license: marshal: %w", err)
	}

	encoded := encoder.EncodeToString(data)

	// Group by 5 characters separated by dashes.
	var groups []string
	for i := 0; i < len(encoded); i += 5 {
		end := i + 5
		if end > len(encoded) {
			end = len(encoded)
		}
		groups = append(groups, encoded[i:end])
	}

	return keyPrefix + strings.Join(groups, "-"), nil
}

// Decode decodes a key string back into a SignedLicense.
func Decode(key string) (SignedLicense, error) {
	if !strings.HasPrefix(key, keyPrefix) {
		return SignedLicense{}, fmt.Errorf("decode license: missing %q prefix", keyPrefix)
	}

	// Remove prefix and dashes.
	raw := strings.TrimPrefix(key, keyPrefix)
	raw = strings.ReplaceAll(raw, "-", "")

	data, err := encoder.DecodeString(raw)
	if err != nil {
		return SignedLicense{}, fmt.Errorf("decode license: base32 decode: %w", err)
	}

	var sl SignedLicense
	if err := json.Unmarshal(data, &sl); err != nil {
		return SignedLicense{}, fmt.Errorf("decode license: unmarshal: %w", err)
	}

	return sl, nil
}
