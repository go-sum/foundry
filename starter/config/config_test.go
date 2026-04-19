package config_test

import (
	"errors"
	"testing"

	"github.com/go-sum/foundry/config"
)

const validCSRFHex = "0000000000000000000000000000000000000000000000000000000000000001"

func TestLoad_UnsetEnv_UsesProduction_CookieSecureTrue(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)
	t.Setenv("SITE_BASE_URL", "http://example.com")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Env != config.Production {
		t.Errorf("cfg.Env = %q, want %q", cfg.Env, config.Production)
	}
	if !cfg.CSRF.CookieSecure {
		t.Error("cfg.CSRF.CookieSecure = false, want true for production")
	}
}

func TestLoad_Development_CookieSecureFalse(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)
	t.Setenv("SITE_BASE_URL", "http://example.com")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Env != config.Development {
		t.Errorf("cfg.Env = %q, want %q", cfg.Env, config.Development)
	}
	if cfg.CSRF.CookieSecure {
		t.Error("cfg.CSRF.CookieSecure = true, want false for development")
	}
}

func TestLoad_Testing_ReturnsTestingEnv(t *testing.T) {
	t.Setenv("APP_ENV", "testing")
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Env != config.Testing {
		t.Errorf("cfg.Env = %q, want %q", cfg.Env, config.Testing)
	}
}

func TestLoad_UnknownEnv_PassesThroughWithNoOverlay(t *testing.T) {
	t.Setenv("APP_ENV", "staging")
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)
	t.Setenv("SITE_BASE_URL", "http://example.com")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Env != "staging" {
		t.Errorf("cfg.Env = %q, want %q", cfg.Env, "staging")
	}
	if !cfg.CSRF.CookieSecure {
		t.Error("cfg.CSRF.CookieSecure = false, want true (no overlay applied for unknown env)")
	}
}

func TestLoad_MissingCSRFKey_AllEnvs_ReturnsError(t *testing.T) {
	for _, env := range []string{"", "production", "development", "testing"} {
		name := env
		if name == "" {
			name = "unset"
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv("APP_ENV", env)
			t.Setenv("SECURITY_CSRF_KEY", "")
			t.Setenv("SECURITY_CSRF_KEY_PREVIOUS", "")

			_, err := config.Load()
			if err == nil {
				t.Fatalf("expected validation error for env=%q, got nil", env)
			}
			if !errors.Is(err, config.ErrCSRFKeyMissing) {
				t.Errorf("got %v; want errors.Is ErrCSRFKeyMissing", err)
			}
		})
	}
}

func TestLoad_MalformedCSRFKey_ReturnsError(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SECURITY_CSRF_KEY", "not-hex")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for malformed SECURITY_CSRF_KEY, got nil")
	}
	if !errors.Is(err, config.ErrCSRFKeyInvalid) {
		t.Errorf("got %v; want errors.Is ErrCSRFKeyInvalid", err)
	}
}

func TestLoad_MissingCSRFKey_ErrorMentionsEnvVar(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SECURITY_CSRF_KEY", "")
	t.Setenv("SECURITY_CSRF_KEY_PREVIOUS", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing SECURITY_CSRF_KEY, got nil")
	}
	if !errors.Is(err, config.ErrCSRFKeyMissing) {
		t.Errorf("got %v; want errors.Is ErrCSRFKeyMissing", err)
	}
}

func TestLoad_DefaultSessionStore_IsMemory(t *testing.T) {
	t.Setenv("APP_ENV", "testing")
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)
	t.Setenv("SESSION_STORE", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.SessionStore != "memory" {
		t.Errorf("cfg.SessionStore = %q, want %q", cfg.SessionStore, "memory")
	}
}

func TestLoad_SessionStore_Cookie(t *testing.T) {
	t.Setenv("APP_ENV", "testing")
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)
	t.Setenv("SESSION_STORE", "cookie")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.SessionStore != "cookie" {
		t.Errorf("cfg.SessionStore = %q, want %q", cfg.SessionStore, "cookie")
	}
}

func TestLoad_SessionStore_Invalid_ReturnsValidationError(t *testing.T) {
	t.Setenv("APP_ENV", "testing")
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)
	t.Setenv("SESSION_STORE", "redis")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected validation error for invalid SESSION_STORE, got nil")
	}
}

func TestLoad_ValidConfig_EnvFieldSet(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)
	t.Setenv("SITE_BASE_URL", "http://example.com")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Env != config.Development {
		t.Errorf("cfg.Env = %q, want %q", cfg.Env, config.Development)
	}
}
