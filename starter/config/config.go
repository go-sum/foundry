package config

import (
	"fmt"

	cfgpkg "github.com/go-sum/config"
	"github.com/go-sum/web/serve"
	"github.com/go-sum/web/secure"
	"github.com/go-sum/web/session"
	"github.com/go-sum/web/site"
	"github.com/go-sum/web/static"
)

// Config is the complete application configuration.
type Config struct {
	Assets       static.AssetsConfig
	CSP          secure.CSPNonceConfig
	CSRF         secure.CSRFConfig
	Env          Env
	Headers      secure.HeadersConfig
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
		Assets:    assets,
		CSP:       secure.DefaultCSPNonceConfig(),
		CSRF:      csrf,
		Env:       Production,
		Headers:   secure.DefaultHeadersConfig(),
		RateLimit: secure.DefaultRateLimitProfile(),
		Server:    serve.DefaultServerConfig(),
		Session:      session.DefaultSettings(),
		SessionStore: cfgpkg.ExpandEnv("SESSION_STORE", "memory"),
		Site:         site.Config{BaseURL: cfgpkg.ExpandEnv("SITE_BASE_URL", "")},
	}, nil
}
