package provider

import (
	"testing"
	"time"
)

func TestApplyDefaults_ZeroConfig(t *testing.T) {
	cfg := ApplyDefaults(Config{})

	if cfg.CodeTTL != 5*time.Minute {
		t.Errorf("CodeTTL = %v, want %v", cfg.CodeTTL, 5*time.Minute)
	}
	if cfg.AccessTokenTTL != time.Hour {
		t.Errorf("AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, time.Hour)
	}
	if cfg.RefreshTokenTTL != 30*24*time.Hour {
		t.Errorf("RefreshTokenTTL = %v, want %v", cfg.RefreshTokenTTL, 30*24*time.Hour)
	}
	if !cfg.RequirePKCE {
		t.Error("RequirePKCE = false, want true")
	}
}

func TestApplyDefaults_NonZeroTTLsPreserved(t *testing.T) {
	input := Config{
		CodeTTL:         10 * time.Minute,
		AccessTokenTTL:  2 * time.Hour,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}
	cfg := ApplyDefaults(input)

	if cfg.CodeTTL != 10*time.Minute {
		t.Errorf("CodeTTL = %v, want %v", cfg.CodeTTL, 10*time.Minute)
	}
	if cfg.AccessTokenTTL != 2*time.Hour {
		t.Errorf("AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, 2*time.Hour)
	}
	if cfg.RefreshTokenTTL != 7*24*time.Hour {
		t.Errorf("RefreshTokenTTL = %v, want %v", cfg.RefreshTokenTTL, 7*24*time.Hour)
	}
}

func TestApplyDefaults_RequirePKCEAlwaysTrue(t *testing.T) {
	// Even when explicitly set to false, RequirePKCE must be forced to true.
	cfg := ApplyDefaults(Config{RequirePKCE: false})
	if !cfg.RequirePKCE {
		t.Error("RequirePKCE = false after ApplyDefaults, want true")
	}

	cfg2 := ApplyDefaults(Config{RequirePKCE: true})
	if !cfg2.RequirePKCE {
		t.Error("RequirePKCE = false after ApplyDefaults with true input, want true")
	}
}

func TestApplyDefaults_NegativeTTLsGetDefaults(t *testing.T) {
	input := Config{
		CodeTTL:         -1 * time.Second,
		AccessTokenTTL:  -1 * time.Second,
		RefreshTokenTTL: -1 * time.Second,
	}
	cfg := ApplyDefaults(input)

	if cfg.CodeTTL != 5*time.Minute {
		t.Errorf("CodeTTL = %v, want %v", cfg.CodeTTL, 5*time.Minute)
	}
	if cfg.AccessTokenTTL != time.Hour {
		t.Errorf("AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, time.Hour)
	}
	if cfg.RefreshTokenTTL != 30*24*time.Hour {
		t.Errorf("RefreshTokenTTL = %v, want %v", cfg.RefreshTokenTTL, 30*24*time.Hour)
	}
}

func TestApplyDefaults_IssuerPreserved(t *testing.T) {
	input := Config{Issuer: "https://auth.example.com"}
	cfg := ApplyDefaults(input)
	if cfg.Issuer != "https://auth.example.com" {
		t.Errorf("Issuer = %q, want %q", cfg.Issuer, "https://auth.example.com")
	}
}
