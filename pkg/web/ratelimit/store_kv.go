package ratelimit

import (
	"context"
	"fmt"
	"time"
)

// KVStoreConfig controls how rate-limit buckets are keyed in the backing KV store.
type KVStoreConfig struct {
	Prefix string
	Now    func() time.Time
}

// KVStore is the minimal rate-limit-specific contract required by the KV-backed store.
// It is consumer-owned by pkg/web/ratelimit rather than producer-owned by the generic KV module.
type KVStore interface {
	RateLimitAllow(ctx context.Context, key string, capacity int, refillPer time.Duration, now time.Time) (allowed bool, retryAfter time.Duration, remaining int, resetAfter time.Duration, err error)
}

// kvStore persists rate-limit buckets in a shared KV store.
type kvStore struct {
	kvs    KVStore
	prefix string
	now    func() time.Time
}

// DefaultKVStoreConfig returns the production default key namespace for KV-backed rate limits.
func DefaultKVStoreConfig() KVStoreConfig {
	return KVStoreConfig{Prefix: "ratelimit:"}
}

// NewKVStore builds a Store over a rate-limit-capable KV store.
func NewKVStore(kvs KVStore, cfg KVStoreConfig) Store {
	if kvs == nil {
		panic("web/ratelimit: KVStore must not be nil")
	}
	if cfg.Prefix == "" {
		cfg.Prefix = DefaultKVStoreConfig().Prefix
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &kvStore{
		kvs:    kvs,
		prefix: cfg.Prefix,
		now:    cfg.Now,
	}
}

// Allow implements Store.
func (s *kvStore) Allow(ctx context.Context, key string, policy Policy) (Decision, error) {
	if err := policy.Validate(); err != nil {
		return Decision{}, err
	}
	allowed, retryAfter, remaining, resetAfter, err := s.kvs.RateLimitAllow(ctx, s.key(key), policy.Capacity, policy.RefillPer, s.now())
	if err != nil {
		return Decision{}, fmt.Errorf("web/ratelimit: kv store allow: %w", err)
	}
	return Decision{
		Allowed:    allowed,
		RetryAfter: retryAfter,
		Limit:      policy.Capacity,
		Remaining:  remaining,
		ResetAfter: resetAfter,
	}, nil
}

func (s *kvStore) key(key string) string {
	return s.prefix + key
}
