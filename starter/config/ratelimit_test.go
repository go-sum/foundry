package config_test

import (
	"testing"
	"time"

	"github.com/go-sum/foundry/config"
)

func TestDefaultRateLimitsConfig_PoliciesAreValid(t *testing.T) {
	cfg := config.DefaultRateLimitsConfig()

	if cfg.Auth.Capacity < 1 {
		t.Errorf("Auth.Capacity = %d, want >= 1", cfg.Auth.Capacity)
	}
	if cfg.Auth.RefillPer <= 0 {
		t.Errorf("Auth.RefillPer = %v, want > 0", cfg.Auth.RefillPer)
	}
	if cfg.ContactSubmit.Capacity < 1 {
		t.Errorf("ContactSubmit.Capacity = %d, want >= 1", cfg.ContactSubmit.Capacity)
	}
	if cfg.ContactSubmit.RefillPer <= 0 {
		t.Errorf("ContactSubmit.RefillPer = %v, want > 0", cfg.ContactSubmit.RefillPer)
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
	if authPolicy.Capacity != cfg.Auth.Capacity {
		t.Errorf("auth policy capacity = %d, want %d", authPolicy.Capacity, cfg.Auth.Capacity)
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
