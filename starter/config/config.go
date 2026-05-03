package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/go-sum/foundry/pkg/componentry/interactive/theme"
	cfgpkg "github.com/go-sum/foundry/pkg/config"
	"github.com/go-sum/foundry/pkg/db"
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
	Secure                  secure.SecureConfig
	Server                  serve.ServerConfig
	Session                 session.Settings
	SessionStore            string `validate:"oneof=memory cookie kv" help:"set SESSION_STORE to memory, cookie, or kv"`
	AllowMemorySessionStore bool   `help:"set SESSION_STORE_ALLOW_MEMORY=true only for tests or intentionally ephemeral runtimes"`
}

// Config is the complete application configuration.
type Config struct {
	App       AppConfig
	Assets    static.AssetsConfig
	Auth      AuthConfig
	DB        db.DBConfig
	KV        cfgpkg.KVConfig
	LogLevel  string
	RateLimit RateLimitsConfig
	Site      site.Config
	Web       WebConfig
}

func Load() (*Config, error) {
	return cfgpkg.Load(cfgpkg.LoadParams[Config]{
		Base: productionConfig,
		Rules: func(cfg Config) []func(*validator.Validate) {
			return []func(*validator.Validate){
				site.ValidationRules(),
				serve.ValidationRules(),
				session.ValidationRules(cfg.Web.SessionStore, cfg.KV.Password, cfg.Web.Session.CookieKey, cfg.Web.AllowMemorySessionStore),
				emailProviderRules(cfg.App.Email.Provider),
				rateLimitStoreRules(cfg.RateLimit.Store.Type),
			}
		},
	})
}

// WorkerConfig holds only the configuration needed by the background worker.
// The web process uses the full Config; the worker uses this leaner struct so
// it does not require web-only secrets (CSRF key, session key, KV URL).
type WorkerConfig struct {
	App      AppConfig
	DB       db.DBConfig
	LogLevel string
}

// LoadWorker loads only the configuration needed by the background worker.
func LoadWorker() (*WorkerConfig, error) {
	return cfgpkg.Load(cfgpkg.LoadParams[WorkerConfig]{
		Base: workerConfig,
		Rules: func(cfg WorkerConfig) []func(*validator.Validate) {
			return []func(*validator.Validate){
				emailProviderRules(cfg.App.Email.Provider),
			}
		},
	})
}

func workerConfig() (WorkerConfig, error) {
	appCfg, err := productionApp()
	if err != nil {
		return WorkerConfig{}, err
	}
	return WorkerConfig{
		App:      appCfg,
		DB:       db.DBConfig{DSN: cfgpkg.ExpandSecret("DATABASE_URL")},
		LogLevel: cfgpkg.ExpandEnv("LOG_LEVEL", "info"),
	}, nil
}

func productionConfig() (Config, error) {
	siteBaseURL := cfgpkg.ExpandEnv("SITE_BASE_URL", "")
	csrfAllowMissingOrigin, err := cfgpkg.ExpandEnvBool("SECURITY_CSRF_ALLOW_MISSING_ORIGIN", false)
	if err != nil {
		return Config{}, err
	}
	csrfCookieSecure, err := cfgpkg.ExpandEnvBool("SECURITY_CSRF_COOKIE_SECURE", true)
	if err != nil {
		return Config{}, err
	}
	sessionCookieSecure, err := cfgpkg.ExpandEnvBool("SESSION_COOKIE_SECURE", true)
	if err != nil {
		return Config{}, err
	}
	allowMemorySessionStore, err := cfgpkg.ExpandEnvBool("SESSION_STORE_ALLOW_MEMORY", false)
	if err != nil {
		return Config{}, err
	}
	appCfg, err := productionApp()
	if err != nil {
		return Config{}, err
	}
	assetsCfg := static.InitialAssetsConfig(cfgpkg.ExpandEnv("ASSETS_PUBLIC_DIR", ""))
	assetsCfg.URLPrefix = cfgpkg.ExpandEnv("ASSETS_URL_PREFIX", assetsCfg.URLPrefix)

	headers := secure.InitialHeadersConfig()
	headers.CrossOriginEmbedderPolicy = cfgpkg.ExpandEnv("SECURITY_HEADERS_COEP", headers.CrossOriginEmbedderPolicy)
	headers.CrossOriginOpenerPolicy = cfgpkg.ExpandEnv("SECURITY_HEADERS_COOP", headers.CrossOriginOpenerPolicy)

	csp := secure.InitialCSPNonceConfig().WithScriptHashes(theme.InitScriptCSPHash)
	if hashes := cfgpkg.ExpandEnvCSV("SECURITY_CSP_EXTRA_SCRIPT_HASHES"); len(hashes) > 0 {
		csp = csp.WithScriptHashes(hashes...)
	}
	csrf := secure.CSRFConfigFromHex(cfgpkg.ExpandSecret("SECURITY_CSRF_KEY"))
	csrf.AllowMissingOrigin = csrfAllowMissingOrigin
	csrf.CookieSecure = csrfCookieSecure

	sessionCfg := session.InitialSessionSettings(cfgpkg.ExpandEnv("SESSION_KV_PREFIX", ""))
	sessionCfg.CookieKey = session.CookieKeyFromHex(cfgpkg.ExpandSecret("SECURITY_SESSION_KEY"))
	sessionCfg.CookieSecure = sessionCookieSecure

	authCfg := productionAuth(siteBaseURL)

	return Config{
		App:    appCfg,
		Assets: assetsCfg,
		Auth:   authCfg,
		DB: db.DBConfig{
			DSN: cfgpkg.ExpandSecret("DATABASE_URL"),
		},
		KV:        cfgpkg.ParseKVURL(cfgpkg.ExpandSecret("KV_URL")),
		LogLevel:  cfgpkg.ExpandEnv("LOG_LEVEL", "info"),
		RateLimit: productionRateLimits(cfgpkg.ExpandEnv("RATELIMIT_STORE", "kv")),
		Site: site.Config{
			BaseURL:      siteBaseURL,
			AllowedHosts: site.BuildAllowedHosts(siteBaseURL, cfgpkg.ExpandEnv("SITE_ALLOWED_HOSTS", "")),
		},
		Web: WebConfig{
			Secure: secure.SecureConfig{
				CSP:     csp,
				CSRF:    csrf,
				Headers: headers,
			},
			Server:                  serve.ServerConfigFromEnv(),
			Session:                 sessionCfg,
			SessionStore:            cfgpkg.ExpandEnv("SESSION_STORE", "cookie"),
			AllowMemorySessionStore: allowMemorySessionStore,
		},
	}, nil
}
