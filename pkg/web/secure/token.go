// Package secure provides security middleware for the web package:
// CSRF protection, security headers, rate limiting, and CORS.
package secure

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"time"
)

// Token and key errors.
var (
	ErrTokenInvalid = errors.New("secure: invalid token")
	ErrTokenExpired = errors.New("secure: expired token")
	// ErrKeyTooShort is returned by SignURL when the provided key is shorter than
	// the minimum required length of 32 bytes.
	ErrKeyTooShort = errors.New("secure: key must be at least 32 bytes")
)

// tokenSize is the wire format: 16 nonce + 8 iat + 8 exp + 32 mac = 64 bytes.
const tokenSize = 64

// IssueToken creates an HMAC-SHA256 signed, time-limited token.
//
// Wire format (64 bytes, base64url-encoded):
//
//	[0:16]  random nonce
//	[16:24] issued-at (unix seconds, big-endian)
//	[24:32] expiry (unix seconds, big-endian)
//	[32:64] HMAC-SHA256(key, scope || 0x00 || nonce || iat || exp)
func IssueToken(key []byte, scope string, ttl time.Duration) (string, error) {
	now := time.Now()
	exp := now.Add(ttl)

	raw := make([]byte, tokenSize)

	// Nonce.
	if _, err := rand.Read(raw[:16]); err != nil {
		return "", err
	}

	// Timestamps.
	binary.BigEndian.PutUint64(raw[16:24], uint64(now.Unix()))
	binary.BigEndian.PutUint64(raw[24:32], uint64(exp.Unix()))

	// MAC.
	mac := computeMAC(key, scope, raw[:32])
	copy(raw[32:64], mac)

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// VerifyToken validates an HMAC-SHA256 signed token.
// Returns nil if valid, ErrTokenInvalid if malformed/tampered,
// or ErrTokenExpired if past expiry.
func VerifyToken(key []byte, scope string, encoded string) error {
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(raw) != tokenSize {
		return ErrTokenInvalid
	}

	// Verify MAC first (before checking expiry to prevent timing attacks).
	expected := computeMAC(key, scope, raw[:32])
	if !hmac.Equal(raw[32:64], expected) {
		return ErrTokenInvalid
	}

	// Check expiry.
	exp := time.Unix(int64(binary.BigEndian.Uint64(raw[24:32])), 0)
	if time.Now().After(exp) {
		return ErrTokenExpired
	}

	return nil
}

func computeMAC(key []byte, scope string, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(scope))
	h.Write([]byte{0x00})
	h.Write(data)
	return h.Sum(nil)
}
