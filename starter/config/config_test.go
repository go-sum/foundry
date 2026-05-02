package config_test

import (
	"strings"
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
	if !cfg.Web.Secure.CSRF.CookieSecure {
		t.Error("cfg.Web.Secure.CSRF.CookieSecure = false, want true for production")
	}
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
	if !cfg.Web.Secure.CSRF.CookieSecure {
		t.Error("cfg.Web.Secure.CSRF.CookieSecure = false, want true (no overlay applied for unknown env)")
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
			t.Setenv("SECURITY_AUTH_TOKEN_KEY", validAuthTokenHex)
			t.Setenv("SITE_BASE_URL", "http://example.com") // required in production

			_, err := config.Load()
			if err == nil {
				t.Fatalf("expected validation error for env=%q, got nil", env)
			}
			if !strings.Contains(err.Error(), "CSRF") {
				t.Errorf("got %v; want error mentioning CSRF", err)
			}
		})
	}
}

func TestLoad_MalformedCSRFKey_ReturnsError(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SITE_BASE_URL", "http://example.com")
	t.Setenv("SECURITY_CSRF_KEY", "not-hex")
	t.Setenv("SECURITY_AUTH_TOKEN_KEY", validAuthTokenHex)

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for malformed SECURITY_CSRF_KEY, got nil")
	}
	if !strings.Contains(err.Error(), "CSRF") {
		t.Errorf("got %v; want error mentioning CSRF", err)
	}
}

func TestLoad_MissingCSRFKey_ErrorMentionsEnvVar(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SITE_BASE_URL", "http://example.com")
	t.Setenv("SECURITY_CSRF_KEY", "")
	t.Setenv("SECURITY_AUTH_TOKEN_KEY", validAuthTokenHex)

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing SECURITY_CSRF_KEY, got nil")
	}
	if !strings.Contains(err.Error(), "SECURITY_CSRF_KEY") {
		t.Errorf("got %v; want error mentioning SECURITY_CSRF_KEY", err)
	}
}

func TestLoad_MissingAuthTokenKey_ReturnsError(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SITE_BASE_URL", "http://example.com")
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)
	t.Setenv("SECURITY_AUTH_TOKEN_KEY", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing SECURITY_AUTH_TOKEN_KEY, got nil")
	}
	if !strings.Contains(err.Error(), "Secrets") {
		t.Errorf("got %v; want error mentioning Secrets", err)
	}
}

func TestLoad_MalformedAuthTokenKey_ReturnsError(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SITE_BASE_URL", "http://example.com")
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)
	t.Setenv("SECURITY_AUTH_TOKEN_KEY", "not-hex")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for malformed SECURITY_AUTH_TOKEN_KEY, got nil")
	}
	if !strings.Contains(err.Error(), "Secrets") {
		t.Errorf("got %v; want error mentioning Secrets", err)
	}
}

func TestLoad_AuthTokenKey_TooShort_ReturnsError(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SITE_BASE_URL", "http://example.com")
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)
	// 31 bytes = 62 hex chars — one byte short of the 32-byte minimum
	t.Setenv("SECURITY_AUTH_TOKEN_KEY", "0000000000000000000000000000000000000000000000000000000000001")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for short SECURITY_AUTH_TOKEN_KEY, got nil")
	}
	if !strings.Contains(err.Error(), "Secrets") {
		t.Errorf("got %v; want error mentioning Secrets", err)
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

func TestLoad_Production_MissingBaseURL_ReturnsError(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	setValidSecrets(t)
	t.Setenv("SITE_BASE_URL", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing SITE_BASE_URL in production, got nil")
	}
	if !strings.Contains(err.Error(), "BaseURL") {
		t.Errorf("got %v; want error mentioning BaseURL", err)
	}
}

func TestLoad_DefaultSessionStore_IsCookie(t *testing.T) {
	t.Setenv("APP_ENV", "testing")
	setValidSecrets(t)
	t.Setenv("SESSION_STORE", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Web.SessionStore != "cookie" {
		t.Errorf("cfg.Web.SessionStore = %q, want %q", cfg.Web.SessionStore, "cookie")
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
	if cfg.Web.SessionStore != "cookie" {
		t.Errorf("cfg.Web.SessionStore = %q, want %q", cfg.Web.SessionStore, "cookie")
	}
}

func TestLoad_SessionStore_KV(t *testing.T) {
	t.Setenv("APP_ENV", "testing")
	setValidSecrets(t)
	t.Setenv("SESSION_STORE", "kv")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Web.SessionStore != "kv" {
		t.Errorf("cfg.Web.SessionStore = %q, want %q", cfg.Web.SessionStore, "kv")
	}
}

func TestLoad_SessionStore_KV_RequiresPasswordOutsideTesting(t *testing.T) {
	for _, env := range []string{"production", "development"} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("APP_ENV", env)
			setValidSecrets(t)
			t.Setenv("SITE_BASE_URL", "https://example.com")
			t.Setenv("SESSION_STORE", "kv")
			t.Setenv("KV_URL", "redis://kv:6379") // URL without password

			_, err := config.Load()
			if err == nil {
				t.Fatal("expected error for missing password in KV_URL, got nil")
			}
			if !strings.Contains(err.Error(), "SessionStore") {
				t.Fatalf("got %v, want error mentioning SessionStore", err)
			}
		})
	}
}

func TestLoad_KVTLS_Enabled(t *testing.T) {
	t.Setenv("APP_ENV", "testing")
	setValidSecrets(t)
	t.Setenv("KV_URL", "rediss://kv:6379")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.KV.TLSEnabled {
		t.Fatal("cfg.KV.TLSEnabled = false, want true")
	}
}

func TestLoad_KVURL_Parsed(t *testing.T) {
	t.Setenv("APP_ENV", "testing")
	setValidSecrets(t)
	t.Setenv("KV_URL", "redis://:mypass@kv:6379")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.KV.Addr != "kv:6379" {
		t.Errorf("cfg.KV.Addr = %q, want %q", cfg.KV.Addr, "kv:6379")
	}
	if cfg.KV.Password != "mypass" {
		t.Errorf("cfg.KV.Password = %q, want %q", cfg.KV.Password, "mypass")
	}
	if cfg.KV.TLSEnabled {
		t.Error("cfg.KV.TLSEnabled = true, want false for redis:// scheme")
	}
}

func TestLoad_KVURL_DefaultsWhenEmpty(t *testing.T) {
	t.Setenv("APP_ENV", "testing")
	setValidSecrets(t)
	t.Setenv("KV_URL", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.KV.Addr != "localhost:6379" {
		t.Errorf("cfg.KV.Addr = %q, want %q", cfg.KV.Addr, "localhost:6379")
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

func TestLoad_SessionStore_Memory_OnlyAllowedInTesting(t *testing.T) {
	for _, env := range []string{"production", "development"} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("APP_ENV", env)
			setValidSecrets(t)
			t.Setenv("SITE_BASE_URL", "https://example.com")
			t.Setenv("SESSION_STORE", "memory")

			_, err := config.Load()
			if err == nil {
				t.Fatal("expected error for memory session store outside testing, got nil")
			}
			if !strings.Contains(err.Error(), "SessionStore") {
				t.Fatalf("got %v, want error mentioning SessionStore", err)
			}
		})
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
	// Default base URL is http://foundry.test, so hostname should be foundry.test.
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

func TestLoad_Production_AllowedHosts_FromBaseURLAndEnv(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	setValidSecrets(t)
	t.Setenv("SITE_BASE_URL", "https://example.com")
	t.Setenv("SITE_ALLOWED_HOSTS", "www.example.com, cdn.example.com")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	wantHosts := []string{"example.com", "www.example.com", "cdn.example.com"}
	if got, want := len(cfg.Site.AllowedHosts), len(wantHosts); got != want {
		t.Fatalf("AllowedHosts length = %d, want %d (got %v)", got, want, cfg.Site.AllowedHosts)
	}
	for i, h := range wantHosts {
		if cfg.Site.AllowedHosts[i] != h {
			t.Errorf("AllowedHosts[%d] = %q, want %q", i, cfg.Site.AllowedHosts[i], h)
		}
	}
}

func TestLoad_Production_AllowedHosts_BaseURLOnly(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	setValidSecrets(t)
	t.Setenv("SITE_BASE_URL", "https://example.com")
	t.Setenv("SITE_ALLOWED_HOSTS", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := len(cfg.Site.AllowedHosts), 1; got != want {
		t.Fatalf("AllowedHosts length = %d, want %d (got %v)", got, want, cfg.Site.AllowedHosts)
	}
	if cfg.Site.AllowedHosts[0] != "example.com" {
		t.Errorf("AllowedHosts[0] = %q, want %q", cfg.Site.AllowedHosts[0], "example.com")
	}
}

func TestLoad_Production_AllowedHosts_Empty_ReturnsError(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	setValidSecrets(t)
	// Empty BaseURL yields empty AllowedHosts; both the required tag on BaseURL
	// and the AllowedHosts production struct-level validator fire together.
	t.Setenv("SITE_BASE_URL", "")
	t.Setenv("SITE_ALLOWED_HOSTS", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for empty AllowedHosts in production, got nil")
	}
	if !strings.Contains(err.Error(), "AllowedHosts") {
		t.Errorf("Load() error = %v, want error mentioning AllowedHosts", err)
	}
}

func TestLoad_ServerTrustedProxies_FromEnv(t *testing.T) {
	t.Setenv("APP_ENV", "testing")
	setValidSecrets(t)
	t.Setenv("SERVER_TRUSTED_PROXIES", " 192.0.2.0/24 , 10.0.0.0/8 ,, ")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := len(cfg.Web.Server.TrustedProxies), 2; got != want {
		t.Fatalf("TrustedProxies length = %d, want %d", got, want)
	}
	if got := cfg.Web.Server.TrustedProxies[0]; got != "192.0.2.0/24" {
		t.Errorf("TrustedProxies[0] = %q, want %q", got, "192.0.2.0/24")
	}
	if got := cfg.Web.Server.TrustedProxies[1]; got != "10.0.0.0/8" {
		t.Errorf("TrustedProxies[1] = %q, want %q", got, "10.0.0.0/8")
	}
}

func TestLoad_ServerTrustedProxies_InvalidCIDR_ReturnsError(t *testing.T) {
	t.Setenv("APP_ENV", "testing")
	setValidSecrets(t)
	t.Setenv("SERVER_TRUSTED_PROXIES", "192.0.2.0/24,not-a-cidr")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid SERVER_TRUSTED_PROXIES, got nil")
	}
	if !strings.Contains(err.Error(), "TrustedProxies") {
		t.Errorf("error = %v, want error mentioning TrustedProxies", err)
	}
}

// Development overlay: security posture must match production except for the
// documented Air SSE workaround (COEP/COOP cleared).

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

// Testing overlay: documents the intentional relaxations applied to the test
// environment. Each test below ensures the override is deliberate and catches
// any future accidental removal.

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
