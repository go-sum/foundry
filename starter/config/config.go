package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/go-sum/foundry/pkg/auth/provider"
	"github.com/go-sum/foundry/pkg/componentry/interactive/theme"
	cfgpkg "github.com/go-sum/foundry/pkg/config"
	"github.com/go-sum/foundry/pkg/db"
	"github.com/go-sum/foundry/pkg/notification/email"
	"github.com/go-sum/foundry/pkg/web/ratelimit"
	"github.com/go-sum/foundry/pkg/web/secure"
	"github.com/go-sum/foundry/pkg/web/serve"
	"github.com/go-sum/foundry/pkg/web/session"
	"github.com/go-sum/foundry/pkg/web/site"
	"github.com/go-sum/foundry/pkg/web/static"
)

// WebConfig is application-level glue that composes independent pkg/web sub-package
// configs. It lives here rather than in pkg/web because each sub-package owns its own
// config type; this struct adds the app-specific SessionStore policy that determines
// which session store implementation gets wired up.
type WebConfig struct {
	Secure       secure.SecureConfig
	Server       serve.ServerConfig
	Session      session.Settings
	SessionStore string `validate:"oneof=memory cookie kv" help:"set SESSION_STORE to memory (testing only), cookie, or kv"`
}

// Config is the complete application configuration.
type Config struct {
	App       AppConfig
	Assets    static.AssetsConfig
	Auth      AuthConfig
	DB        db.DBConfig
	Env       Env
	KV        cfgpkg.KVConfig
	LogLevel  string
	RateLimit RateLimitsConfig
	Site      site.Config
	Web       WebConfig
}

type Env string

const (
	Production  Env = "production"
	Development Env = "development"
	Testing     Env = "testing"
)

func loadParams() cfgpkg.LoadParams[Config] {
	return cfgpkg.LoadParams[Config]{
		Base: func() (Config, error) { return productionConfig(), nil },
		Env:  cfgpkg.ExpandEnv("APP_ENV", string(Production)),
		Overlays: []cfgpkg.EnvOverlay[Config]{
			{Name: string(Development), Apply: developmentOverlay},
			{Name: string(Testing), Apply: testingOverlay},
		},
		SetEnv: func(c *Config, e string) { c.Env = Env(e) },
	}
}

func Load() (*Config, error) {
	p := loadParams()
	p.Rules = func(cfg Config) []func(*validator.Validate) {
		return []func(*validator.Validate){
			site.ValidationRules(string(cfg.Env)),
			serve.ValidationRules(),
			session.ValidationRules(cfg.Web.SessionStore, string(cfg.Env), cfg.KV.Password, cfg.Web.Session.CookieKey),
			emailProviderRules(cfg.App.Email.Provider, string(cfg.Env)),
		}
	}
	return cfgpkg.Load[Config](p)
}

func productionConfig() Config {
	siteBaseURL := cfgpkg.ExpandEnv("SITE_BASE_URL", "")
	return Config{
		App:    productionApp(),
		Assets: static.InitialAssetsConfig(""),
		Auth:   productionAuth(siteBaseURL),
		DB: db.DBConfig{
			DSN: cfgpkg.ExpandSecret("DATABASE_URL"),
		},
		Env:       Production,
		KV:        cfgpkg.ParseKVURL(cfgpkg.ExpandSecret("KV_URL")),
		LogLevel:  cfgpkg.ExpandEnv("LOG_LEVEL", "info"),
		RateLimit: productionRateLimits(),
		Site: site.Config{
			BaseURL:      siteBaseURL,
			AllowedHosts: site.BuildAllowedHosts(siteBaseURL, cfgpkg.ExpandEnv("SITE_ALLOWED_HOSTS", "")),
		},
		Web: WebConfig{
			Secure: secure.SecureConfig{
				CSP:     secure.InitialCSPNonceConfig().WithScriptHashes(theme.InitScriptCSPHash),
				CSRF:    secure.CSRFConfigFromHex(cfgpkg.ExpandSecret("SECURITY_CSRF_KEY")),
				Headers: secure.InitialHeadersConfig(),
			},
			Server: serve.ServerConfigFromEnv(),
			Session: func() session.Settings {
				s := session.InitialSessionSettings(cfgpkg.ExpandEnv("SESSION_KV_PREFIX", ""))
				s.CookieKey = session.CookieKeyFromHex(cfgpkg.ExpandSecret("SECURITY_SESSION_KEY"))
				return s
			}(),
			SessionStore: cfgpkg.ExpandEnv("SESSION_STORE", "cookie"),
		},
	}
}

func developmentOverlay(cfg *Config) {
	if cfg.Site.BaseURL == "" {
		// Caddy serves the dev domain over HTTPS (tls internal / mkcert).
		cfg.Site.BaseURL = "https://foundry.test"
		// BaseURL changed after productionConfig built AllowedHosts; rebuild.
		cfg.Site.AllowedHosts = site.BuildAllowedHosts(cfg.Site.BaseURL, "")
	}
	cfg.Auth.Provider.Issuer = cfg.Site.BaseURL
	cfg.Auth.OAuthClient = provider.BuildOAuthClient(cfg.Site.BaseURL, cfg.Auth.OAuthClient.ClientID, "/auth/callback")
	// Air's built-in reverse proxy rewrites Host to localhost:<app_port>.
	// Allow loopback so AllowedHosts validation passes for dev requests.
	cfg.Site.AllowedHosts = append(cfg.Site.AllowedHosts, "localhost", "127.0.0.1")
	// COEP/COOP enable cross-origin isolation, which causes Chrome to enforce
	// CORS on the service worker's same-origin SSE fetch to Air's proxy. Air
	// does not set Access-Control-Allow-Origin, so the browser drops the
	// connection immediately. Clear only these two headers; all other security
	// settings remain at production defaults.
	cfg.Web.Secure.Headers.CrossOriginEmbedderPolicy = ""
	cfg.Web.Secure.Headers.CrossOriginOpenerPolicy = ""
	cfg.Web.Server.Addr = ":3000"
	// The SHA-256 hash of Air's injected live-reload script; update when .versions:APP_VERSION changes.
	// (github.com/air-verse/air v1.65.0 runner/proxy.js).
	cfg.Web.Secure.CSP = cfg.Web.Secure.CSP.WithScriptHashes("'sha256-y933zYNvpVe5f9j5A+OKECUXiWo8bKB5Yp5sLDD3d0I='")
	cfg.LogLevel = cfgpkg.ExpandEnv("LOG_LEVEL", "debug")
	cfg.App.Email.Provider = email.ProviderLog
}

func testingOverlay(cfg *Config) {
	cfg.Site.BaseURL = "http://test.local"
	cfg.Auth.Provider.Issuer = cfg.Site.BaseURL
	cfg.Auth.OAuthClient = provider.BuildOAuthClient(cfg.Site.BaseURL, cfg.Auth.OAuthClient.ClientID, "/auth/callback")
	cfg.Web.Secure.CSRF.AllowMissingOrigin = true
	cfg.Web.Secure.CSRF.CookieSecure = false
	cfg.Web.Session.CookieSecure = false
	cfg.RateLimit.Store.Type = ratelimit.StoreTypeMemory
	cfg.App.Email.Provider = email.ProviderLog
	if dir := cfgpkg.ExpandEnv("TEST_STATIC_DIR", ""); dir != "" {
		cfg.Assets.PublicDir = dir
	}
}
