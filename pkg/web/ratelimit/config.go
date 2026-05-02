package ratelimit

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/go-sum/foundry/pkg/web"
)

// Policy defines the bucket capacity and token refill cadence for a profile.
// Capacity is the maximum number of tokens in the bucket.
// RefillPer is the time required to replenish one token.
type Policy struct {
	Capacity  int           `validate:"gte=1"`
	RefillPer time.Duration `validate:"gt=0"`
}

func (p Policy) validate() error {
	switch {
	case p.Capacity < 1:
		return fmt.Errorf("%w: capacity must be >= 1", ErrPolicyInvalid)
	case p.RefillPer <= 0:
		return fmt.Errorf("%w: refill period must be > 0", ErrPolicyInvalid)
	default:
		return nil
	}
}

// Validate reports whether the policy is usable.
func (p Policy) Validate() error {
	return p.validate()
}

// RateLimitProfile is a stable profile name bound to a limiter policy.
type RateLimitProfile string

// RateLimitConfig binds a stable profile name to its token-bucket policy.
type RateLimitConfig struct {
	Profile RateLimitProfile `validate:"required"`
	Policy  Policy           `validate:"required"`
}

// RateLimitProfiles is a configured set of named rate-limit profiles.
type RateLimitProfiles []RateLimitConfig

// KeyFunc derives the limiter key for a request.
type KeyFunc func(c *web.Context) (string, error)

// Config builds a Limiter from a shared store and named profiles.
type Config struct {
	Store    Store
	Profiles map[RateLimitProfile]Policy
	Logger   *slog.Logger
}

// MiddlewareConfig configures request-time rate limiting by named profile.
type MiddlewareConfig struct {
	Limiter    *Limiter
	Profile    RateLimitProfile
	KeyFunc    KeyFunc
	FailClosed bool
	OnError    func(err error, c *web.Context)
	Skipper    func(c *web.Context) bool
}

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

// InitialStoreConfig returns an empty StoreConfig. Type must be set by the caller.
func InitialStoreConfig() StoreConfig {
	return StoreConfig{}
}

// NewLimiter builds a Store from storeCfg, then constructs a Limiter with profiles and logger.
func NewLimiter(storeCfg StoreConfig, profiles map[RateLimitProfile]Policy, logger *slog.Logger) (*Limiter, error) {
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
