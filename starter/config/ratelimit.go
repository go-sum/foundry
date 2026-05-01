package config

import (
	"cmp"
	"time"

	"github.com/go-sum/foundry/pkg/web/ratelimit"
)

const (
	RateLimitRoutesAuth         ratelimit.RateLimitProfile = "routes.auth"
	RateLimitContactSubmitEmail ratelimit.RateLimitProfile = "contact.submit.email"
)

// RateLimitStoreConfig selects and tunes the backing rate-limit store.
// Type is overridden per-environment in env.go (testing uses "memory").
type RateLimitStoreConfig struct {
	Type   string // "kv" or "memory"; defaults to "kv" in production
	Prefix string // key namespace; defaults to "ratelimit:" when empty
}

// RateLimitsConfig is the typed rate-limit configuration for the starter.
// Named profile fields mirror the RouteConfig pattern: each field is a
// complete profile spec; the constants above define the canonical names.
type RateLimitsConfig struct {
	Store         RateLimitStoreConfig      `validate:"required"`
	Auth          ratelimit.RateLimitConfig `validate:"required"`
	ContactSubmit ratelimit.RateLimitConfig `validate:"required"`
}

// Profiles converts the typed config to the map form required by ratelimit.New.
func (c RateLimitsConfig) Profiles() map[string]ratelimit.Policy {
	return map[string]ratelimit.Policy{
		string(c.Auth.Profile):          c.Auth.Policy,
		string(c.ContactSubmit.Profile): c.ContactSubmit.Policy,
	}
}

// DefaultRateLimitsConfig returns the production-ready defaults for all profiles.
func DefaultRateLimitsConfig() RateLimitsConfig {
	return applyRateLimitsDefaults(RateLimitsConfig{})
}

func applyRateLimitsDefaults(c RateLimitsConfig) RateLimitsConfig {
	return RateLimitsConfig{
		Store: cmp.Or(c.Store, RateLimitStoreConfig{Type: ratelimit.StoreTypeKV}),
		Auth: cmp.Or(c.Auth, ratelimit.RateLimitConfig{
			Profile: RateLimitRoutesAuth,
			Policy:  ratelimit.Policy{Capacity: 20, RefillPer: 3 * time.Second},
		}),
		ContactSubmit: cmp.Or(c.ContactSubmit, ratelimit.RateLimitConfig{
			Profile: RateLimitContactSubmitEmail,
			Policy:  ratelimit.Policy{Capacity: 3, RefillPer: 20 * time.Minute},
		}),
	}
}
