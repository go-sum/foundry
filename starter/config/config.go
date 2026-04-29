package config

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/go-sum/foundry/pkg/componentry/interactive/theme"
	cfgpkg "github.com/go-sum/foundry/pkg/config"
	"github.com/go-sum/foundry/pkg/web/secure"
	"github.com/go-sum/foundry/pkg/web/serve"
	"github.com/go-sum/foundry/pkg/web/session"
	"github.com/go-sum/foundry/pkg/web/site"
	"github.com/go-sum/foundry/pkg/web/static"
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
	PublicDir    string
	Assets       static.AssetsConfig
	Auth         AuthConfig
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

	if cfg.Env == Production && cfg.Site.BaseURL == "" {
		return nil, ErrBaseURLMissing
	}

	if cfg.Env == Production && len(cfg.Site.AllowedHosts) == 0 {
		return nil, ErrAllowedHostsEmpty
	}

	if err := cfgpkg.Validate(cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return &cfg, nil
}

func defaultProduction() (Config, error) {
	siteBaseURL := cfgpkg.ExpandEnv("SITE_BASE_URL", "")
	csrf, err := defaultCSRF()
	if err != nil {
		return Config{}, fmt.Errorf("config: security: %w", err)
	}
	serverCfg, err := serve.DefaultServerConfigFromEnv()
	if err != nil {
		return Config{}, err
	}
	authCfg, err := DefaultAuth(siteBaseURL)
	if err != nil {
		return Config{}, fmt.Errorf("config: auth: %w", err)
	}
	const publicDir = "public"
	assets := static.DefaultAssetsConfig()
	assets.PublicDir = filepath.Join(publicDir, "static")

	return Config{
		PublicDir: publicDir,
		Assets:    assets,
		Auth:      authCfg,
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
		Server:       serverCfg,
		Session:      session.DefaultSettings(),
		SessionStore: cfgpkg.ExpandEnv("SESSION_STORE", "memory"),
		Site: site.Config{
			BaseURL:      siteBaseURL,
			AllowedHosts: site.BuildAllowedHosts(siteBaseURL, cfgpkg.ExpandEnv("SITE_ALLOWED_HOSTS", "")),
		},
	}, nil
}
