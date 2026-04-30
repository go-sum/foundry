package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

const (
	tokenDomainVerify   = "auth/verify/v1"
	tokenDomainIdentity = "auth/identity/v1"
)

// ParseTokenKeys decodes hex-encoded AEAD key material for use as auth token
// secrets. keyHex must be non-empty and decode to at least 32 bytes.
// Returns ErrTokenKeyMissing when keyHex is empty and ErrTokenKeyInvalid
// when the value is malformed or too short.
func ParseTokenKeys(keyHex string) ([][]byte, error) {
	if keyHex == "" {
		return nil, ErrTokenKeyMissing
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenKeyInvalid, err)
	}
	if len(key) < 32 {
		return nil, fmt.Errorf("%w: got %d bytes, need at least 32", ErrTokenKeyInvalid, len(key))
	}
	return [][]byte{key}, nil
}

// DeriveTokenSubkeys derives domain-separated subkeys for email verification
// and identity tokens from the same master key material. A single env var is
// sufficient; the two outputs are cryptographically unrelated.
func DeriveTokenSubkeys(masterKeys [][]byte) (verifyKeys, identityKeys [][]byte, err error) {
	for _, mk := range masterKeys {
		vk, err := deriveHKDF(mk, tokenDomainVerify)
		if err != nil {
			return nil, nil, fmt.Errorf("derive verify key: %w", err)
		}
		ik, err := deriveHKDF(mk, tokenDomainIdentity)
		if err != nil {
			return nil, nil, fmt.Errorf("derive identity key: %w", err)
		}
		verifyKeys = append(verifyKeys, vk)
		identityKeys = append(identityKeys, ik)
	}
	return verifyKeys, identityKeys, nil
}

func deriveHKDF(masterKey []byte, info string) ([]byte, error) {
	r := hkdf.New(sha256.New, masterKey, nil, []byte(info))
	derived := make([]byte, 32)
	if _, err := io.ReadFull(r, derived); err != nil {
		return nil, err
	}
	return derived, nil
}

// TokenCodec encodes and decodes verification tokens.
type TokenCodec interface {
	Encode(VerificationToken) (string, error)
	Decode(string) (VerificationToken, error)
}
