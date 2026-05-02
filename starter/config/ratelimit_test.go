package config

import (
	"testing"
)

func TestProductionRateLimits_PoliciesAreValid(t *testing.T) {
	cfg := productionRateLimits()

	for name, p := range cfg.Policies {
		if p.Capacity < 1 {
			t.Errorf("%s: Capacity = %d, want >= 1", name, p.Capacity)
		}
		if p.RefillPer <= 0 {
			t.Errorf("%s: RefillPer = %v, want > 0", name, p.RefillPer)
		}
	}
}

func TestProductionRateLimits_StoreTypeIsKV(t *testing.T) {
	cfg := productionRateLimits()
	if cfg.Store.Type != "kv" {
		t.Errorf("Store.Type = %q, want %q", cfg.Store.Type, "kv")
	}
}
