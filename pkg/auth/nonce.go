package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/go-sum/foundry/pkg/kv"
)

// TokenNonceStore tracks consumed verification tokens to prevent replay attacks.
type TokenNonceStore interface {
	HasConsumed(ctx context.Context, key string) (bool, error)
	MarkConsumed(ctx context.Context, key string, ttl time.Duration) error
}

type kvTokenNonceStore struct {
	store kv.Store
}

// NewKVTokenNonceStore returns a TokenNonceStore backed by a kv.Store.
func NewKVTokenNonceStore(s kv.Store) TokenNonceStore {
	return &kvTokenNonceStore{store: s}
}

func (k *kvTokenNonceStore) HasConsumed(ctx context.Context, key string) (bool, error) {
	n, err := k.store.Exists(ctx, key)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (k *kvTokenNonceStore) MarkConsumed(ctx context.Context, key string, ttl time.Duration) error {
	return k.store.Set(ctx, key, []byte("1"), kv.SetOptions{TTL: ttl})
}

// tokenNonceKey returns a namespaced KV key derived from the raw token ciphertext.
// Hashing avoids storing the full ciphertext as a key.
func tokenNonceKey(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return "auth:nonce:" + hex.EncodeToString(h[:])
}
