package ratelimit

import "fmt"

// StoreConfig specifies how to construct a ratelimit.Store.
type StoreConfig struct {
	// Type selects the backing store: "kv" or "memory".
	Type string
	// Env is the application environment name. The "memory" store type is only
	// permitted when Env equals TestingEnv; any other environment returns an error.
	Env string
	// TestingEnv is the environment name that permits the memory store.
	// Defaults to "testing" if empty.
	TestingEnv string
	// KVBackend is required when Type is "kv".
	KVBackend KVBackend
	// KVPrefix is an optional key prefix applied to all KV bucket keys.
	KVPrefix string
}

// StoreType constants for use with NewStoreFromConfig.
const (
	StoreTypeKV     = "kv"
	StoreTypeMemory = "memory"
)

// NewStoreFromConfig constructs a Store from cfg.
func NewStoreFromConfig(cfg StoreConfig) (Store, error) {
	testingEnv := cfg.TestingEnv
	if testingEnv == "" {
		testingEnv = "testing"
	}

	switch cfg.Type {
	case StoreTypeKV:
		if cfg.KVBackend == nil {
			return nil, fmt.Errorf("ratelimit: kv store requires a non-nil KVBackend")
		}
		return NewKVStore(cfg.KVBackend, KVStoreConfig{Prefix: cfg.KVPrefix}), nil
	case StoreTypeMemory:
		if cfg.Env != testingEnv {
			return nil, fmt.Errorf("ratelimit: memory store is only permitted in the %q environment, got %q", testingEnv, cfg.Env)
		}
		return NewMemoryStore(MemoryStoreConfig{}), nil
	default:
		return nil, fmt.Errorf("ratelimit: unsupported store type %q", cfg.Type)
	}
}
