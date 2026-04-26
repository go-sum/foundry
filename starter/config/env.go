package config

import (
	"os"

	cfgpkg "github.com/go-sum/config"
)

type Env string

const (
	Production  Env = "production"
	Development Env = "development"
	Testing     Env = "testing"
)

var envOverlays = map[Env]func(*Config){
	Development: overlayDevelopment,
	Testing:     overlayTesting,
}

// LoadEnv builds the base production config, resolves the current environment,
// applies its overlay, and returns the fully initialized Config.
func LoadEnv() (Config, error) {
	cfg, err := defaultProduction()
	if err != nil {
		return Config{}, err
	}
	env := Env(cfgpkg.ExpandEnv("APP_ENV", string(Production)))
	if apply, ok := envOverlays[env]; ok {
		apply(&cfg)
	}
	cfg.Env = env
	return cfg, nil
}

// airProxyCSPHash is the SHA-256 hash of Air's injected live-reload script
// (github.com/air-verse/air v1.65.0 runner/proxy.js). Update this constant
// when the Air version in .versions changes.
const airProxyCSPHash = "'sha256-y933zYNvpVe5f9j5A+OKECUXiWo8bKB5Yp5sLDD3d0I='"

func overlayDevelopment(cfg *Config) {
	cfg.Site.BaseURL = "http://forge.test"
	cfg.Auth.Provider.Issuer = cfg.Site.BaseURL
	cfg.CSRF.CookieSecure = false
	cfg.Session.CookieSecure = false
	cfg.Headers.StrictTransportSecurity = ""
	// COEP/COOP enable cross-origin isolation, which causes Chrome to enforce
	// CORS on the service worker's same-origin SSE fetch to Air's proxy. Air
	// does not set Access-Control-Allow-Origin, so the browser drops the
	// connection immediately. These headers are not needed in development.
	cfg.Headers.CrossOriginEmbedderPolicy = ""
	cfg.Headers.CrossOriginOpenerPolicy = ""
	cfg.Server.Addr = ":3000"
	cfg.CSP = cfg.CSP.WithScriptHashes(airProxyCSPHash)
}

func overlayTesting(cfg *Config) {
	cfg.Site.BaseURL = "http://test.local"
	cfg.Auth.Provider.Issuer = cfg.Site.BaseURL
	cfg.CSRF.CookieSecure = false
	cfg.Session.CookieSecure = false
	if dir := os.Getenv("TEST_STATIC_DIR"); dir != "" {
		cfg.Assets.PublicDir = dir
	}
}
