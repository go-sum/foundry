package config_test

import (
	"errors"
	"testing"

	"github.com/go-sum/foundry/config"
)

const (
	validCSRFHex      = "0000000000000000000000000000000000000000000000000000000000000001"
	validAuthTokenHex = "0000000000000000000000000000000000000000000000000000000000000002"
)

func setValidSecrets(t *testing.T) {
	t.Helper()
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)
	t.Setenv("SECURITY_AUTH_TOKEN_KEY", validAuthTokenHex)
}

func TestLoad_UnsetEnv_UsesProduction_CookieSecureTrue(t *testing.T) {
	t.Setenv("APP_ENV", "")
	setValidSecrets(t)
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
	setValidSecrets(t)
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
	setValidSecrets(t)

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
	setValidSecrets(t)
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
			t.Setenv("SECURITY_AUTH_TOKEN_KEY", validAuthTokenHex)

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
	t.Setenv("SECURITY_AUTH_TOKEN_KEY", validAuthTokenHex)

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
	t.Setenv("SECURITY_AUTH_TOKEN_KEY", validAuthTokenHex)

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing SECURITY_CSRF_KEY, got nil")
	}
	if !errors.Is(err, config.ErrCSRFKeyMissing) {
		t.Errorf("got %v; want errors.Is ErrCSRFKeyMissing", err)
	}
}

func TestLoad_MissingAuthTokenKey_ReturnsError(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)
	t.Setenv("SECURITY_AUTH_TOKEN_KEY", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing SECURITY_AUTH_TOKEN_KEY, got nil")
	}
	if !errors.Is(err, config.ErrAuthTokenKeyMissing) {
		t.Errorf("got %v; want errors.Is ErrAuthTokenKeyMissing", err)
	}
}

func TestLoad_MalformedAuthTokenKey_ReturnsError(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)
	t.Setenv("SECURITY_AUTH_TOKEN_KEY", "not-hex")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for malformed SECURITY_AUTH_TOKEN_KEY, got nil")
	}
	if !errors.Is(err, config.ErrAuthTokenKeyInvalid) {
		t.Errorf("got %v; want errors.Is ErrAuthTokenKeyInvalid", err)
	}
}

func TestLoad_AuthTokenKey_TooShort_ReturnsError(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)
	// 31 bytes = 62 hex chars — one byte short of the 32-byte minimum
	t.Setenv("SECURITY_AUTH_TOKEN_KEY", "0000000000000000000000000000000000000000000000000000000000001")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for short SECURITY_AUTH_TOKEN_KEY, got nil")
	}
	if !errors.Is(err, config.ErrAuthTokenKeyInvalid) {
		t.Errorf("got %v; want errors.Is ErrAuthTokenKeyInvalid", err)
	}
}

func TestLoad_AuthTokenKeys_Populated(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	setValidSecrets(t)
	t.Setenv("SITE_BASE_URL", "http://example.com")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Auth.TokenKeys) == 0 {
		t.Error("cfg.Auth.TokenKeys is empty, want at least one key")
	}
	if len(cfg.Auth.TokenKeys[0]) < 32 {
		t.Errorf("cfg.Auth.TokenKeys[0] length = %d, want >= 32", len(cfg.Auth.TokenKeys[0]))
	}
}

func TestLoad_DefaultSessionStore_IsMemory(t *testing.T) {
	t.Setenv("APP_ENV", "testing")
	setValidSecrets(t)
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
	setValidSecrets(t)
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
	setValidSecrets(t)
	t.Setenv("SESSION_STORE", "redis")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected validation error for invalid SESSION_STORE, got nil")
	}
}

func TestLoad_ValidConfig_EnvFieldSet(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	setValidSecrets(t)
	t.Setenv("SITE_BASE_URL", "http://example.com")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Env != config.Development {
		t.Errorf("cfg.Env = %q, want %q", cfg.Env, config.Development)
	}
}
