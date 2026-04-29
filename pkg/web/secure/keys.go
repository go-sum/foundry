package secure

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// decodeHexKey decodes a hex-encoded key string and returns the raw bytes.
// Trims surrounding whitespace. Returns an error if the input is not valid
// hex or if the decoded key is shorter than 32 bytes.
func decodeHexKey(s string) ([]byte, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return nil, err
	}
	if len(raw) < 32 {
		return nil, ErrKeyTooShort
	}
	return raw, nil
}

// GenerateKeyHex generates a random 32-byte key and returns it as a 64-char
// hex string suitable for use as SECURITY_CSRF_KEY.
func GenerateKeyHex() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
