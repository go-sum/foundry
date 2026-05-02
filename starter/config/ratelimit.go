package config

import (
	"time"

	"github.com/go-sum/foundry/pkg/web/ratelimit"
)

const (
	RateLimitRoutesAuth         ratelimit.RateLimitProfile = "routes.auth"
	RateLimitContactSubmitEmail ratelimit.RateLimitProfile = "contact.submit.email"
)

// RateLimitsConfig is the typed rate-limit configuration for the starter.
// Profile names are code-owned constants; only the Policy (capacity, refill
// rate) is operator-configurable.
type RateLimitsConfig struct {
	Store         ratelimit.StoreConfig `validate:"required"`
	Auth          ratelimit.Policy      `validate:"required"`
	ContactSubmit ratelimit.Policy      `validate:"required"`
}

// Profiles converts the typed config to the map form required by ratelimit.New.
func (c RateLimitsConfig) Profiles() map[string]ratelimit.Policy {
	return map[string]ratelimit.Policy{
		string(RateLimitRoutesAuth):         c.Auth,
		string(RateLimitContactSubmitEmail): c.ContactSubmit,
	}
}

// DefaultRateLimitsConfig returns the production-ready defaults for all profiles.
func DefaultRateLimitsConfig() RateLimitsConfig {
	return RateLimitsConfig{
		Store:         ratelimit.StoreConfig{Type: ratelimit.StoreTypeKV},
		Auth:          ratelimit.Policy{Capacity: 20, RefillPer: 3 * time.Second},
		ContactSubmit: ratelimit.Policy{Capacity: 3, RefillPer: 20 * time.Minute},
	}
}
