package ratelimit

import (
	"fmt"
	"log/slog"
)

// StoreConfig specifies how to construct a ratelimit.Store.
type StoreConfig struct {
	// Type selects the backing store: "kv" or "memory".
	Type string
	// KVStore is required when Type is "kv".
	KVStore KVStore
	// KVPrefix is an optional key prefix applied to all KV bucket keys.
	KVPrefix string
}

// StoreType constants for use with NewStoreFromConfig.
const (
	StoreTypeKV     = "kv"
	StoreTypeMemory = "memory"
)

// WithKVStore returns a copy of s with KVStore set from v when v implements KVStore.
// Passing nil or a value that does not implement KVStore is a no-op.
func (s StoreConfig) WithKVStore(v any) StoreConfig {
	if kvs, ok := v.(KVStore); ok {
		s.KVStore = kvs
	}
	return s
}

// NewLimiter builds a Store from storeCfg, then constructs a Limiter with profiles and logger.
func NewLimiter(storeCfg StoreConfig, profiles map[string]Policy, logger *slog.Logger) (*Limiter, error) {
	store, err := NewStoreFromConfig(storeCfg)
	if err != nil {
		return nil, err
	}
	return New(Config{Store: store, Profiles: profiles, Logger: logger})
}

// NewStoreFromConfig constructs a Store from cfg.
func NewStoreFromConfig(cfg StoreConfig) (Store, error) {
	switch cfg.Type {
	case StoreTypeKV:
		if cfg.KVStore == nil {
			return nil, fmt.Errorf("ratelimit: kv store requires a non-nil KVStore")
		}
		return NewKVStore(cfg.KVStore, KVStoreConfig{Prefix: cfg.KVPrefix}), nil
	case StoreTypeMemory:
		return NewMemoryStore(MemoryStoreConfig{}), nil
	default:
		return nil, fmt.Errorf("ratelimit: unsupported store type %q", cfg.Type)
	}
}
