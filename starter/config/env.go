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

func overlayDevelopment(cfg *Config) {
	cfg.Site.BaseURL = "http://forge.test"
	cfg.CSRF.CookieSecure = false
	cfg.Session.CookieSecure = false
	cfg.Headers.StrictTransportSecurity = ""
}

func overlayTesting(cfg *Config) {
	cfg.Site.BaseURL = "http://test.local"
	cfg.Session.CookieSecure = false
	if dir := os.Getenv("TEST_STATIC_DIR"); dir != "" {
		cfg.Assets.PublicDir = dir
	}
}
