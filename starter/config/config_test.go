package config_test

import (
	"testing"

	"github.com/go-sum/foundry/config"
)

const (
	validCSRFHex       = "0000000000000000000000000000000000000000000000000000000000000001"
	validAuthTokenHex  = "0000000000000000000000000000000000000000000000000000000000000002"
	validSessionKeyHex = "0000000000000000000000000000000000000000000000000000000000000003"
)

func setValidSecrets(t *testing.T) {
	t.Helper()
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)
	t.Setenv("SECURITY_AUTH_TOKEN_KEY", validAuthTokenHex)
	t.Setenv("SECURITY_SESSION_KEY", validSessionKeyHex)
}

func TestLoad_Development_CookieSecureTrue(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	setValidSecrets(t)
	t.Setenv("SITE_BASE_URL", "https://example.com")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Env != config.Development {
		t.Errorf("cfg.Env = %q, want %q", cfg.Env, config.Development)
	}
	// Dev runs behind Caddy with TLS — secure cookie flags must stay on.
	if !cfg.Web.Secure.CSRF.CookieSecure {
		t.Error("cfg.Web.Secure.CSRF.CookieSecure = false, want true: dev is served over HTTPS via Caddy")
	}
	if !cfg.Web.Session.CookieSecure {
		t.Error("cfg.Web.Session.CookieSecure = false, want true: dev is served over HTTPS via Caddy")
	}
}

func TestLoad_Development_CSRF_AllowMissingOrigin_False(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	setValidSecrets(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	// Browsers on HTTPS always send Sec-Fetch-Site / Origin — disabling this
	// check in dev would mask CSRF origin bugs before they reach production.
	if cfg.Web.Secure.CSRF.AllowMissingOrigin {
		t.Error("cfg.Web.Secure.CSRF.AllowMissingOrigin = true, want false: dev runs over HTTPS and must enforce origin checks")
	}
}

func TestLoad_Development_HSTS_Enabled(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	setValidSecrets(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Web.Secure.Headers.StrictTransportSecurity == "" {
		t.Error("cfg.Web.Secure.Headers.StrictTransportSecurity is empty, want HSTS set: dev is served over HTTPS via Caddy")
	}
}

func TestLoad_Development_COEP_COOP_Cleared(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	setValidSecrets(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	// COEP/COOP are cleared in dev to prevent Chrome from blocking Air's SSE
	// live-reload stream (Air does not set Access-Control-Allow-Origin).
	if cfg.Web.Secure.Headers.CrossOriginEmbedderPolicy != "" {
		t.Errorf("cfg.Web.Secure.Headers.CrossOriginEmbedderPolicy = %q, want empty: must be cleared for Air SSE", cfg.Web.Secure.Headers.CrossOriginEmbedderPolicy)
	}
	if cfg.Web.Secure.Headers.CrossOriginOpenerPolicy != "" {
		t.Errorf("cfg.Web.Secure.Headers.CrossOriginOpenerPolicy = %q, want empty: must be cleared for Air SSE", cfg.Web.Secure.Headers.CrossOriginOpenerPolicy)
	}
}

func TestLoad_Testing_CookieSecure_False(t *testing.T) {
	t.Setenv("APP_ENV", "testing")
	setValidSecrets(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	// Tests run over plain HTTP (no TLS); secure cookie flags must be off so
	// cookies are sent and session state works in httptest-based tests.
	if cfg.Web.Secure.CSRF.CookieSecure {
		t.Error("cfg.Web.Secure.CSRF.CookieSecure = true, want false: tests run over plain HTTP")
	}
	if cfg.Web.Session.CookieSecure {
		t.Error("cfg.Web.Session.CookieSecure = true, want false: tests run over plain HTTP")
	}
}

func TestLoad_Testing_AllowMissingOrigin_True(t *testing.T) {
	t.Setenv("APP_ENV", "testing")
	setValidSecrets(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	// httptest requests do not set Origin or Sec-Fetch-Site headers; allowing
	// missing origin prevents CSRF middleware from rejecting all test requests.
	// Security-specific tests that exercise origin validation set these headers
	// explicitly and use their own configs.
	if !cfg.Web.Secure.CSRF.AllowMissingOrigin {
		t.Error("cfg.Web.Secure.CSRF.AllowMissingOrigin = false, want true: httptest requests omit Origin headers")
	}
}

func TestLoad_Development_SiteBaseURL_DefaultsToForgeTest(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	setValidSecrets(t)
	t.Setenv("SITE_BASE_URL", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	// Default dev domain uses https:// — Caddy terminates TLS with tls internal.
	if cfg.Site.BaseURL != "https://foundry.test" {
		t.Errorf("Site.BaseURL = %q, want %q", cfg.Site.BaseURL, "https://foundry.test")
	}
}

func TestLoad_Development_SiteBaseURL_RespectedWhenSet(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	setValidSecrets(t)
	t.Setenv("SITE_BASE_URL", "http://starter.test")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Site.BaseURL != "http://starter.test" {
		t.Errorf("Site.BaseURL = %q, want %q", cfg.Site.BaseURL, "http://starter.test")
	}
	if cfg.Auth.Provider.Issuer != "http://starter.test" {
		t.Errorf("Auth.Provider.Issuer = %q, want %q", cfg.Auth.Provider.Issuer, "http://starter.test")
	}
	// AllowedHosts should include base URL hostname + Air proxy hosts.
	wantHosts := []string{"starter.test", "localhost", "127.0.0.1"}
	if got, want := len(cfg.Site.AllowedHosts), len(wantHosts); got != want {
		t.Fatalf("AllowedHosts length = %d, want %d (got %v)", got, want, cfg.Site.AllowedHosts)
	}
	for i, h := range wantHosts {
		if cfg.Site.AllowedHosts[i] != h {
			t.Errorf("AllowedHosts[%d] = %q, want %q", i, cfg.Site.AllowedHosts[i], h)
		}
	}
}

func TestLoad_Development_AllowedHosts_DefaultBaseURL_IncludesForgeTest(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	setValidSecrets(t)
	t.Setenv("SITE_BASE_URL", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	// Default base URL is https://foundry.test, so hostname should be foundry.test.
	wantHosts := []string{"foundry.test", "localhost", "127.0.0.1"}
	if got, want := len(cfg.Site.AllowedHosts), len(wantHosts); got != want {
		t.Fatalf("AllowedHosts length = %d, want %d (got %v)", got, want, cfg.Site.AllowedHosts)
	}
	for i, h := range wantHosts {
		if cfg.Site.AllowedHosts[i] != h {
			t.Errorf("AllowedHosts[%d] = %q, want %q", i, cfg.Site.AllowedHosts[i], h)
		}
	}
}

func TestLoad_SessionKVPrefix_Override(t *testing.T) {
	t.Setenv("APP_ENV", "testing")
	setValidSecrets(t)
	t.Setenv("SESSION_KV_PREFIX", "starter-a:session:")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Web.Session.KVPrefix != "starter-a:session:" {
		t.Fatalf("cfg.Web.Session.KVPrefix = %q, want %q", cfg.Web.Session.KVPrefix, "starter-a:session:")
	}
}
