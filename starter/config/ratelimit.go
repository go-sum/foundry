package config

import (
	"time"

	"github.com/go-sum/foundry/pkg/web/ratelimit"
)

const (
	RateLimitRoutesAuth         ratelimit.RateLimitProfile = "routes.auth"
	RateLimitContactSubmitEmail ratelimit.RateLimitProfile = "contact.submit.email"
)

type RateLimitsConfig struct {
	Store    ratelimit.StoreConfig `validate:"required"`
	Policies map[ratelimit.RateLimitProfile]ratelimit.Policy
}

func productionRateLimits() RateLimitsConfig {
	return RateLimitsConfig{
		Store: ratelimit.StoreConfig{Type: ratelimit.StoreTypeKV},
		Policies: map[ratelimit.RateLimitProfile]ratelimit.Policy{
			RateLimitRoutesAuth:         {Capacity: 20, RefillPer: 3 * time.Second},
			RateLimitContactSubmitEmail: {Capacity: 3, RefillPer: 20 * time.Minute},
		},
	}
}
