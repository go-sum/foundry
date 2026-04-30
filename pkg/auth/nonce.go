package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// TokenNonceStore tracks consumed verification tokens to prevent replay attacks.
type TokenNonceStore interface {
	HasConsumed(ctx context.Context, key string) (bool, error)
	MarkConsumed(ctx context.Context, key string, ttl time.Duration) error
}

// tokenNonceKey returns a namespaced KV key derived from the raw token ciphertext.
// Hashing avoids storing the full ciphertext as a key.
func tokenNonceKey(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return "auth:nonce:" + hex.EncodeToString(h[:])
}
