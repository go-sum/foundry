package config

import (
	"time"

	"github.com/go-playground/validator/v10"
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

func productionRateLimits(storeType string) RateLimitsConfig {
	return RateLimitsConfig{
		Store: ratelimit.StoreConfig{Type: storeType},
		Policies: map[ratelimit.RateLimitProfile]ratelimit.Policy{
			RateLimitRoutesAuth:         {Capacity: 20, RefillPer: 3 * time.Second},
			RateLimitContactSubmitEmail: {Capacity: 3, RefillPer: 20 * time.Minute},
		},
	}
}

func rateLimitStoreRules(storeType string) func(*validator.Validate) {
	return func(v *validator.Validate) {
		v.RegisterStructValidation(func(sl validator.StructLevel) {
			if storeType != ratelimit.StoreTypeKV && storeType != ratelimit.StoreTypeMemory {
				sl.ReportError(storeType, "Store", "Store", "oneof", "kv memory")
			}
		}, RateLimitsConfig{})
	}
}
