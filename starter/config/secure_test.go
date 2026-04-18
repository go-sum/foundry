package config_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-sum/foundry/config"
)

const validHexKey = "0000000000000000000000000000000000000000000000000000000000000001"

func TestLoad_ValidHexKey_PopulatesKey(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SECURITY_CSRF_KEY", validHexKey)
	t.Setenv("SECURITY_CSRF_KEY_PREVIOUS", "")
	t.Setenv("SITE_BASE_URL", "http://example.com")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.CSRF.Key) != 32 {
		t.Errorf("len(cfg.CSRF.Key) = %d, want 32", len(cfg.CSRF.Key))
	}
}

func TestLoad_WithPreviousKeys_PopulatesList(t *testing.T) {
	prevKey := strings.Repeat("ab", 32)
	t.Setenv("APP_ENV", "production")
	t.Setenv("SECURITY_CSRF_KEY", validHexKey)
	t.Setenv("SECURITY_CSRF_KEY_PREVIOUS", prevKey)
	t.Setenv("SITE_BASE_URL", "http://example.com")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.CSRF.PreviousKeys) != 1 {
		t.Errorf("len(PreviousKeys) = %d, want 1", len(cfg.CSRF.PreviousKeys))
	}
	if len(cfg.CSRF.PreviousKeys[0]) != 32 {
		t.Errorf("len(PreviousKeys[0]) = %d, want 32", len(cfg.CSRF.PreviousKeys[0]))
	}
}

func TestLoad_MalformedPreviousKey_ReturnsError(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SECURITY_CSRF_KEY", validHexKey)
	t.Setenv("SECURITY_CSRF_KEY_PREVIOUS", "not-hex")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for malformed SECURITY_CSRF_KEY_PREVIOUS, got nil")
	}
	if !errors.Is(err, config.ErrCSRFPrevKeysInvalid) {
		t.Errorf("got %v; want errors.Is ErrCSRFPrevKeysInvalid", err)
	}
}
