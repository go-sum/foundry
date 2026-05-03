package session

import (
	"cmp"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/cookiecodec"
)

// Settings is the env-facing shape for session configuration.
type Settings struct {
	CookieName   string `validate:"required"`
	IdleTTL      time.Duration
	AbsoluteTTL  time.Duration
	CookieSecure bool
	KVPrefix     string
	CookieKey    []byte
}

// Config configures the session Middleware.
type Config struct {
	// Store handles session persistence. Required.
	// Use NewMemoryStore for test-only server-side sessions.
	// Use NewKVStore for server-side sessions.
	// Use NewCookieStore for client-side AEAD-encrypted sessions.
	Store Store `validate:"required"`

	// CookieTemplate defines the attributes of the session cookie.
	// CookieTemplate.Name is required.
	CookieTemplate web.Cookie

	// TTL is the absolute session lifetime. Defaults to 24 hours.
	TTL time.Duration

	// IdleTTL is the idle-inactivity timeout. Zero disables idle expiry.
	IdleTTL time.Duration

	// MaxCookieBytes is the maximum serialized Set-Cookie size. Defaults to 4096.
	MaxCookieBytes int
}

// StoreConfig specifies how to construct a session.Store.
// Build the appropriate fields based on Type before calling NewStoreFromConfig.
type StoreConfig struct {
	// Type selects the backing store: "cookie", "kv", or "memory".
	Type string
	// AllowMemory explicitly permits the in-memory store type. Use only in
	// tests or other intentionally non-production runtimes.
	AllowMemory bool
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

// KVStoreConfig controls how server-side sessions are keyed in the backing KV
// store.
type KVStoreConfig struct {
	Prefix string
}

const (
	defaultTTL     = 24 * time.Hour
	defaultMaxSize = 4096
)

// StoreType constants for use with NewStoreFromConfig.
const (
	StoreTypeCookie = "cookie"
	StoreTypeKV     = "kv"
	StoreTypeMemory = "memory"
)

// CookieKeyFromHex decodes a hex-encoded session cookie key.
// Trims whitespace, hex-decodes, and checks that the key is at least 32 bytes.
// Returns nil on empty input, invalid hex, or a key shorter than 32 bytes.
func CookieKeyFromHex(keyHex string) []byte {
	s := strings.TrimSpace(keyHex)
	if s == "" {
		return nil
	}
	raw, err := hex.DecodeString(s)
	if err != nil {
		return nil
	}
	if len(raw) < 32 {
		return nil
	}
	return raw
}

// InitialSessionSettings returns production-grade session defaults.
// kvPrefix overrides the default KV key prefix; pass "" to use the default.
func InitialSessionSettings(kvPrefix string) Settings {
	kvCfg := InitialKVStoreConfig()
	return Settings{
		CookieName:   "session",
		IdleTTL:      30 * time.Minute,
		AbsoluteTTL:  24 * time.Hour,
		CookieSecure: true,
		KVPrefix:     cmp.Or(kvPrefix, kvCfg.Prefix),
	}
}

// CookieCodecFromSettings constructs the AEAD cookie codec for cookie-backed
// sessions using the configured cookie name and key material.
func NewCookieCodec(s Settings) (*cookiecodec.Codec, error) {
	return cookiecodec.New(cookiecodec.Config{
		Name:    s.CookieName,
		Secrets: [][]byte{s.CookieKey},
		Mode:    cookiecodec.AEAD,
	})
}

// NewConfig builds a session Config from Settings and a Store.
func NewConfig(s Settings, store Store) Config {
	return Config{
		Store: store,
		CookieTemplate: web.Cookie{
			Name:     s.CookieName,
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
			Secure:   s.CookieSecure,
		},
		TTL:     s.AbsoluteTTL,
		IdleTTL: s.IdleTTL,
	}
}

// ValidationRules returns a registrar that enforces session store constraints.
// The memory store is only permitted when explicitly enabled; the kv store
// requires a non-empty password; the cookie store requires a key of at least
// 32 bytes.
func ValidationRules(storeType, kvPassword string, cookieKey []byte, allowMemory bool) func(*validator.Validate) {
	return func(v *validator.Validate) {
		v.RegisterStructValidation(func(sl validator.StructLevel) {
			if storeType == StoreTypeMemory && !allowMemory {
				sl.ReportError(storeType, "SessionStore", "SessionStore", "session_store_testing_only", "")
			}
			if storeType == StoreTypeKV && kvPassword == "" {
				sl.ReportError(kvPassword, "SessionStore", "SessionStore", "kv_password_required", "")
			}
			if storeType == StoreTypeCookie && len(cookieKey) < 32 {
				sl.ReportError(cookieKey, "SessionCookieKey", "SessionCookieKey", "session_cookie_key_required", "")
			}
		}, Settings{})
	}
}

// NewStoreFromConfig constructs a session Store and its middleware Config from
// a StoreConfig. It returns an error if required fields are missing or the
// Type/AllowMemory combination is not permitted.
func NewStoreFromConfig(cfg StoreConfig) (Config, Store, error) {
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
		if !cfg.AllowMemory {
			return Config{}, nil, fmt.Errorf("session: memory store requires explicit AllowMemory=true")
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

// InitialKVStoreConfig returns the production default key namespace for KV
// backed sessions.
func InitialKVStoreConfig() KVStoreConfig {
	return KVStoreConfig{Prefix: "session:"}
}
