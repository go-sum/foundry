package config_test

import (
	"testing"

	"github.com/go-sum/foundry/config"
)

const validHexKey = "0000000000000000000000000000000000000000000000000000000000000001"

func TestLoad_ValidHexKey_PopulatesKey(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SECURITY_CSRF_KEY", validHexKey)
	t.Setenv("SECURITY_AUTH_TOKEN_KEY", validAuthTokenHex)
	t.Setenv("SECURITY_SESSION_KEY", validSessionKeyHex)
	t.Setenv("SITE_BASE_URL", "http://example.com")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Web.Secure.CSRF.Key) != 32 {
		t.Errorf("len(cfg.Web.Secure.CSRF.Key) = %d, want 32", len(cfg.Web.Secure.CSRF.Key))
	}
}
