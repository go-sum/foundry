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

func setBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("SECURITY_CSRF_KEY", validCSRFHex)
	t.Setenv("SECURITY_AUTH_TOKEN_KEY", validAuthTokenHex)
	t.Setenv("SECURITY_SESSION_KEY", validSessionKeyHex)
	t.Setenv("SITE_BASE_URL", "https://example.com")
	t.Setenv("EMAIL_PROVIDER", "log")
}

func TestLoad_DefaultsAreProductionSafe(t *testing.T) {
	setBaseEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Web.Server.Addr != ":8080" {
		t.Fatalf("Addr = %q, want %q", cfg.Web.Server.Addr, ":8080")
	}
	if cfg.Web.Secure.CSRF.AllowMissingOrigin {
		t.Fatal("AllowMissingOrigin = true, want false")
	}
	if !cfg.Web.Secure.CSRF.CookieSecure {
		t.Fatal("CSRF CookieSecure = false, want true")
	}
	if !cfg.Web.Session.CookieSecure {
		t.Fatal("Session CookieSecure = false, want true")
	}
	if cfg.Web.Secure.Headers.CrossOriginEmbedderPolicy == "" {
		t.Fatal("COEP = empty, want production default")
	}
	if cfg.Web.Secure.Headers.CrossOriginOpenerPolicy == "" {
		t.Fatal("COOP = empty, want production default")
	}
	if cfg.App.Email.Provider != "log" {
		t.Fatalf("Email.Provider = %q, want %q", cfg.App.Email.Provider, "log")
	}
	if got, want := cfg.Site.AllowedHosts, []string{"example.com"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("AllowedHosts = %v, want %v", got, want)
	}
}

func TestLoad_ExplicitDevStyleOverrides(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("SERVER_ADDR", ":3000")
	t.Setenv("SITE_BASE_URL", "https://foundry.test")
	t.Setenv("SITE_ALLOWED_HOSTS", "localhost,127.0.0.1")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SECURITY_HEADERS_COEP", "")
	t.Setenv("SECURITY_HEADERS_COOP", "")
	t.Setenv("SECURITY_CSP_EXTRA_SCRIPT_HASHES", "'sha256-example'")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Web.Server.Addr != ":3000" {
		t.Fatalf("Addr = %q, want %q", cfg.Web.Server.Addr, ":3000")
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.Web.Secure.Headers.CrossOriginEmbedderPolicy != "" {
		t.Fatalf("COEP = %q, want empty", cfg.Web.Secure.Headers.CrossOriginEmbedderPolicy)
	}
	if cfg.Web.Secure.Headers.CrossOriginOpenerPolicy != "" {
		t.Fatalf("COOP = %q, want empty", cfg.Web.Secure.Headers.CrossOriginOpenerPolicy)
	}
	if !strings.Contains(strings.Join(cfg.Web.Secure.CSP.ScriptSrcExtra, ","), "'sha256-example'") {
		t.Fatalf("ScriptSrcExtra = %v, want extra hash", cfg.Web.Secure.CSP.ScriptSrcExtra)
	}
	if got, want := cfg.Site.AllowedHosts, []string{"foundry.test", "localhost", "127.0.0.1"}; len(got) != len(want) {
		t.Fatalf("AllowedHosts = %v, want %v", got, want)
	}
}

func TestLoad_ExplicitHTTPTestingStyleOverrides(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("SITE_BASE_URL", "http://test.local")
	t.Setenv("SECURITY_CSRF_ALLOW_MISSING_ORIGIN", "true")
	t.Setenv("SECURITY_CSRF_COOKIE_SECURE", "false")
	t.Setenv("SESSION_COOKIE_SECURE", "false")
	t.Setenv("RATELIMIT_STORE", "memory")
	t.Setenv("ASSETS_PUBLIC_DIR", t.TempDir())

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Web.Secure.CSRF.AllowMissingOrigin {
		t.Fatal("AllowMissingOrigin = false, want true")
	}
	if cfg.Web.Secure.CSRF.CookieSecure {
		t.Fatal("CSRF CookieSecure = true, want false")
	}
	if cfg.Web.Session.CookieSecure {
		t.Fatal("Session CookieSecure = true, want false")
	}
	if cfg.RateLimit.Store.Type != "memory" {
		t.Fatalf("RateLimit.Store.Type = %q, want %q", cfg.RateLimit.Store.Type, "memory")
	}
	if cfg.Assets.PublicDir == "" {
		t.Fatal("Assets.PublicDir = empty, want explicit override")
	}
}

func TestLoad_MemorySessionStoreRequiresExplicitAllow(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("SESSION_STORE", "memory")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
}

func TestLoad_MemorySessionStoreAllowedWhenEnabled(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("SESSION_STORE", "memory")
	t.Setenv("SESSION_STORE_ALLOW_MEMORY", "true")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Web.SessionStore != "memory" {
		t.Fatalf("SessionStore = %q, want %q", cfg.Web.SessionStore, "memory")
	}
}

func TestLoad_RejectsEmptyEmailProvider(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("EMAIL_PROVIDER", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
}

func TestLoad_RejectsInvalidEmailProvider(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("EMAIL_PROVIDER", "ses")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
}

func TestLoadWorker_LoadsWithoutWebSecrets(t *testing.T) {
	t.Setenv("EMAIL_PROVIDER", "log")
	// Deliberately NOT setting SECURITY_CSRF_KEY, SECURITY_SESSION_KEY,
	// SECURITY_AUTH_TOKEN_KEY, KV_URL, or SITE_BASE_URL.

	cfg, err := config.LoadWorker()
	if err != nil {
		t.Fatalf("LoadWorker() error = %v", err)
	}
	if cfg.App.Email.Provider != "log" {
		t.Fatalf("Email.Provider = %q, want %q", cfg.App.Email.Provider, "log")
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	// WorkerConfig has no Web, KV, Site, Assets, or Auth fields —
	// their absence is a compile-time guarantee that they are not loaded.
}

func TestLoadWorker_RejectsEmptyEmailProvider(t *testing.T) {
	t.Setenv("EMAIL_PROVIDER", "")

	_, err := config.LoadWorker()
	if err == nil {
		t.Fatal("LoadWorker() error = nil, want validation error")
	}
}

func TestLoadWorker_ReadsLogLevelOverride(t *testing.T) {
	t.Setenv("EMAIL_PROVIDER", "log")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := config.LoadWorker()
	if err != nil {
		t.Fatalf("LoadWorker() error = %v", err)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
}

func TestLoad_AuthIssuerAndClientIDAreDerivedOnce(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("AUTH_ISSUER", "https://auth.example.com")
	t.Setenv("AUTH_FIRST_PARTY_CLIENT_ID", "starter-first-party")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Auth.Provider.Issuer != "https://auth.example.com" {
		t.Fatalf("Provider.Issuer = %q, want %q", cfg.Auth.Provider.Issuer, "https://auth.example.com")
	}
	if cfg.Auth.OAuthClient.Issuer != "https://auth.example.com" {
		t.Fatalf("OAuthClient.Issuer = %q, want %q", cfg.Auth.OAuthClient.Issuer, "https://auth.example.com")
	}
	if cfg.Auth.OAuthClient.ClientID != "starter-first-party" {
		t.Fatalf("OAuthClient.ClientID = %q, want %q", cfg.Auth.OAuthClient.ClientID, "starter-first-party")
	}
	if cfg.Auth.OAuthClient.RedirectURL != "https://auth.example.com/auth/callback" {
		t.Fatalf("OAuthClient.RedirectURL = %q, want %q", cfg.Auth.OAuthClient.RedirectURL, "https://auth.example.com/auth/callback")
	}
}
