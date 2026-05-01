package config

import (
	"fmt"
	"path/filepath"
	"strconv"
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
	Addr       string
	Password   string
	TLSEnabled bool
}

// EmailConfig configures outbound email delivery.
type EmailConfig struct {
	Provider string // "resend" | "mailchannels" | "log"
	APIKey   string
	BaseURL  string
	From     string
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
	Email        EmailConfig
	CSP          secure.CSPNonceConfig
	CSRF         secure.CSRFConfig
	Env          Env
	LogLevel     string
	Headers      secure.HeadersConfig
	KV           KVConfig
	RateLimit    secure.RateLimitProfile
	Server       serve.ServerConfig
	Session      session.Settings
	SessionStore string `validate:"oneof=memory cookie kv"`
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
	if err := validateSessionStore(cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func defaultProduction() (Config, error) {
	siteBaseURL := cfgpkg.ExpandEnv("SITE_BASE_URL", "")
	kvTLS, err := strconv.ParseBool(cfgpkg.ExpandEnv("KV_TLS", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("config: KV_TLS: %w", err)
	}
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
	sessionCfg := session.DefaultSettings()
	if prefix := cfgpkg.ExpandEnv("SESSION_KV_PREFIX", ""); prefix != "" {
		sessionCfg.KVPrefix = prefix
	}
	const publicDir = "public"
	assets := static.DefaultAssetsConfig()
	assets.PublicDir = filepath.Join(publicDir, "static")

	return Config{
		PublicDir: publicDir,
		Assets:    assets,
		Auth:      authCfg,
		Contact: ContactConfig{
			SendTo:     cfgpkg.ExpandEnv("EMAIL_SEND_TO", "send@example.com"),
			SendFrom:   cfgpkg.ExpandEnv("EMAIL_SEND_FROM", "noreply@example.com"),
			RateLimit:  3,
			RateWindow: time.Hour,
		},
		Email: EmailConfig{
			Provider: "log",
			APIKey:   cfgpkg.ExpandSecret("EMAIL_API_KEY"),
			From:     cfgpkg.ExpandEnv("EMAIL_SEND_FROM", "noreply@example.com"),
		},
		CSP:      secure.DefaultCSPNonceConfig().WithScriptHashes(theme.InitScriptCSPHash),
		CSRF:     csrf,
		Env:      Production,
		LogLevel: cfgpkg.ExpandEnv("LOG_LEVEL", "info"),
		Headers:  secure.DefaultHeadersConfig(),
		KV: KVConfig{
			Addr:       cfgpkg.ExpandEnv("KV_HOST", "localhost") + ":" + cfgpkg.ExpandEnv("KV_PORT", "6379"),
			Password:   cfgpkg.ExpandSecret("KV_PASSWORD"),
			TLSEnabled: kvTLS,
		},
		RateLimit:    secure.DefaultRateLimitProfile(),
		Server:       serverCfg,
		Session:      sessionCfg,
		SessionStore: cfgpkg.ExpandEnv("SESSION_STORE", "cookie"),
		Site: site.Config{
			BaseURL:      siteBaseURL,
			AllowedHosts: site.BuildAllowedHosts(siteBaseURL, cfgpkg.ExpandEnv("SITE_ALLOWED_HOSTS", "")),
		},
	}, nil
}

func validateSessionStore(cfg Config) error {
	if cfg.SessionStore == "memory" && cfg.Env != Testing {
		return ErrSessionStoreMemoryTestingOnly
	}
	if cfg.SessionStore == "kv" && cfg.Env != Testing && cfg.KV.Password == "" {
		return ErrKVPasswordMissing
	}
	return nil
}
