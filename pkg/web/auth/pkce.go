package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

// verifierRandomBytes is encoded with unpadded base64url to produce a
// 64-character verifier, comfortably inside RFC 7636's 43-128 character range.
const verifierRandomBytes = 48

var (
	// ErrInvalidVerifier is returned by Challenge when the verifier does not satisfy RFC 7636 4.1.
	ErrInvalidVerifier = errors.New("pkce: verifier does not satisfy RFC 7636 4.1")
)

// NewVerifier generates a cryptographically random PKCE code verifier.
// It is verifierRandomBytes random bytes encoded as base64url (no padding) — 64 chars,
// well within the RFC 7636 43–128 character limit.
func NewVerifier() (string, error) {
	raw := make([]byte, verifierRandomBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func isValidVerifier(v string) bool {
	n := len(v)
	if n < 43 || n > 128 {
		return false
	}
	for i := 0; i < n; i++ {
		c := v[i]
		if (c >= 'A' && c <= 'Z') ||
			(c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') ||
			c == '-' || c == '.' || c == '_' || c == '~' {
			continue
		}
		return false
	}
	return true
}

// Challenge derives the PKCE code_challenge from a verifier using the S256 method:
//
//	challenge = BASE64URL(SHA-256(verifier))
//
// Returns ErrInvalidVerifier if the verifier does not satisfy RFC 7636 4.1.
func Challenge(verifier string) (string, error) {
	if !isValidVerifier(verifier) {
		return "", ErrInvalidVerifier
	}
	// RFC 7636 4.1 restricts code_verifier to unreserved ASCII characters,
	// so converting the string to bytes is an exact ASCII octet mapping.
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}
