package session

import (
	"fmt"

	"github.com/go-sum/foundry/pkg/web/cookiecodec"
)

// StoreType constants for use with NewStoreFromConfig.
const (
	StoreTypeCookie = "cookie"
	StoreTypeKV     = "kv"
	StoreTypeMemory = "memory"
)

// StoreConfig specifies how to construct a session.Store.
// Build the appropriate fields based on Type before calling NewStoreFromConfig.
type StoreConfig struct {
	// Type selects the backing store: "cookie", "kv", or "memory".
	Type string
	// Env is the application environment name. The "memory" store type is only
	// permitted when Env equals TestingEnv; any other environment returns an error.
	Env string
	// TestingEnv is the environment name that permits the memory store.
	// Defaults to "testing" if empty.
	TestingEnv string
	// Settings provides cookie name, TTL, and other session settings.
	Settings Settings
	// Codec is required when Type is "cookie".
	Codec *cookiecodec.Codec
	// KVStore is required when Type is "kv".
	KVStore KVStore
	// KVPrefix is an optional key prefix applied to all KV session keys.
	KVPrefix string
	// TestFactory overrides the memory store constructor. Used in tests to
	// inject a pre-configured store instead of creating a new MemoryStore.
	TestFactory func() Store
}

// NewStoreFromConfig constructs a session Store and its middleware Config from
// a StoreConfig. It returns an error if required fields are missing or the
// Type/Env combination is not permitted.
func NewStoreFromConfig(cfg StoreConfig) (Config, Store, error) {
	testingEnv := cfg.TestingEnv
	if testingEnv == "" {
		testingEnv = "testing"
	}

	var store Store
	switch cfg.Type {
	case StoreTypeCookie:
		if cfg.Codec == nil {
			return Config{}, nil, fmt.Errorf("session: cookie store requires a non-nil Codec")
		}
		store = NewCookieStore(cfg.Codec)
	case StoreTypeKV:
		if cfg.KVStore == nil {
			return Config{}, nil, fmt.Errorf("session: kv store requires a non-nil KVStore")
		}
		store = NewKVStore(cfg.KVStore, KVStoreConfig{Prefix: cfg.KVPrefix})
	case StoreTypeMemory:
		if cfg.Env != testingEnv {
			return Config{}, nil, fmt.Errorf("session: memory store is only permitted in the %q environment, got %q", testingEnv, cfg.Env)
		}
		if cfg.TestFactory != nil {
			store = cfg.TestFactory()
		} else {
			store = NewMemoryStore()
		}
	default:
		return Config{}, nil, fmt.Errorf("session: unsupported store type %q", cfg.Type)
	}

	return NewConfig(cfg.Settings, store), store, nil
}
