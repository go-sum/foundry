package config

import (
	"fmt"
	"time"

	"github.com/go-sum/componentry/interactive/theme"
	cfgpkg "github.com/go-sum/config"
	"github.com/go-sum/web/secure"
	"github.com/go-sum/web/serve"
	"github.com/go-sum/web/session"
	"github.com/go-sum/web/site"
	"github.com/go-sum/web/static"
)

// KVConfig holds connection parameters for the key-value store.
type KVConfig struct {
	Addr     string
	Password string
}

// ContactConfig holds configuration for the contact feature.
type ContactConfig struct {
	SendTo     string
	SendFrom   string
	RateLimit  int
	RateWindow time.Duration
}

// Config is the complete application configuration.
type Config struct {
	Assets       static.AssetsConfig
	Contact      ContactConfig
	CSP          secure.CSPNonceConfig
	CSRF         secure.CSRFConfig
	Env          Env
	Headers      secure.HeadersConfig
	KV           KVConfig
	RateLimit    secure.RateLimitProfile
	Server       serve.ServerConfig
	Session      session.Settings
	SessionStore string `validate:"oneof=memory cookie"`
	Site         site.Config
}

func Load() (*Config, error) {
	cfg, err := LoadEnv()
	if err != nil {
		return nil, err
	}

	if err := cfgpkg.Validate(cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return &cfg, nil
}

func defaultProduction() (Config, error) {
	csrf, err := defaultCSRF()
	if err != nil {
		return Config{}, fmt.Errorf("config: security: %w", err)
	}
	assets := static.DefaultAssetsConfig()
	assets.PublicDir = "public/static"

	return Config{
		Assets: assets,
		Contact: ContactConfig{
			SendTo:     cfgpkg.ExpandEnv("CONTACT_SEND_TO", "admin@example.com"),
			SendFrom:   cfgpkg.ExpandEnv("CONTACT_SEND_FROM", "noreply@example.com"),
			RateLimit:  3,
			RateWindow: time.Hour,
		},
		CSP:          secure.DefaultCSPNonceConfig().WithScriptHashes(theme.InitScriptCSPHash),
		CSRF:         csrf,
		Env:          Production,
		Headers:      secure.DefaultHeadersConfig(),
		KV: KVConfig{
			Addr:     cfgpkg.ExpandEnv("KV_HOST", "localhost") + ":" + cfgpkg.ExpandEnv("KV_PORT", "6379"),
			Password: cfgpkg.ExpandSecret("KV_PASSWORD"),
		},
		RateLimit:    secure.DefaultRateLimitProfile(),
		Server:       serve.DefaultServerConfig(),
		Session:      session.DefaultSettings(),
		SessionStore: cfgpkg.ExpandEnv("SESSION_STORE", "memory"),
		Site:         site.Config{BaseURL: cfgpkg.ExpandEnv("SITE_BASE_URL", "")},
	}, nil
}
