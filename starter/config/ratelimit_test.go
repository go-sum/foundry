package config_test

import (
	"testing"
	"time"

	"github.com/go-sum/foundry/config"
)

func TestDefaultRateLimitsConfig_ContainsRequiredProfiles(t *testing.T) {
	cfg := config.DefaultRateLimitsConfig()

	if cfg.Auth.Profile != config.RateLimitRoutesAuth {
		t.Errorf("Auth.Profile = %q, want %q", cfg.Auth.Profile, config.RateLimitRoutesAuth)
	}
	if cfg.ContactSubmit.Profile != config.RateLimitContactSubmitEmail {
		t.Errorf("ContactSubmit.Profile = %q, want %q", cfg.ContactSubmit.Profile, config.RateLimitContactSubmitEmail)
	}
}

func TestDefaultRateLimitsConfig_PoliciesAreValid(t *testing.T) {
	cfg := config.DefaultRateLimitsConfig()

	if cfg.Auth.Policy.Capacity < 1 {
		t.Errorf("Auth.Policy.Capacity = %d, want >= 1", cfg.Auth.Policy.Capacity)
	}
	if cfg.Auth.Policy.RefillPer <= 0 {
		t.Errorf("Auth.Policy.RefillPer = %v, want > 0", cfg.Auth.Policy.RefillPer)
	}
	if cfg.ContactSubmit.Policy.Capacity < 1 {
		t.Errorf("ContactSubmit.Policy.Capacity = %d, want >= 1", cfg.ContactSubmit.Policy.Capacity)
	}
	if cfg.ContactSubmit.Policy.RefillPer <= 0 {
		t.Errorf("ContactSubmit.Policy.RefillPer = %v, want > 0", cfg.ContactSubmit.Policy.RefillPer)
	}
}

func TestRateLimitsConfig_Profiles_ReturnsCorrectMap(t *testing.T) {
	cfg := config.DefaultRateLimitsConfig()
	profiles := cfg.Profiles()

	if len(profiles) != 2 {
		t.Fatalf("Profiles() len = %d, want 2", len(profiles))
	}
	authPolicy, ok := profiles[string(config.RateLimitRoutesAuth)]
	if !ok {
		t.Fatalf("Profiles() missing %q", config.RateLimitRoutesAuth)
	}
	if authPolicy.Capacity != cfg.Auth.Policy.Capacity {
		t.Errorf("auth policy capacity = %d, want %d", authPolicy.Capacity, cfg.Auth.Policy.Capacity)
	}

	contactPolicy, ok := profiles[string(config.RateLimitContactSubmitEmail)]
	if !ok {
		t.Fatalf("Profiles() missing %q", config.RateLimitContactSubmitEmail)
	}
	if contactPolicy.RefillPer != 20*time.Minute {
		t.Errorf("contact policy refill_per = %v, want %v", contactPolicy.RefillPer, 20*time.Minute)
	}
}

func TestDefaultRateLimitsConfig_StoreTypeIsKV(t *testing.T) {
	cfg := config.DefaultRateLimitsConfig()
	if cfg.Store.Type != "kv" {
		t.Errorf("Store.Type = %q, want %q", cfg.Store.Type, "kv")
	}
}
