package app

import (
	"context"
	"fmt"

	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/web/cookiecodec"
	"github.com/go-sum/foundry/pkg/web/ratelimit"
	"github.com/go-sum/foundry/pkg/web/session"

	config "github.com/go-sum/foundry/config"
)


func provideSecurity(_ context.Context, runtime Runtime, kvStore kv.Store, storeFactory func() session.Store) (Security, session.Store, error) {
	cfg := runtime.Config

	origins := make([]string, 0, 1+len(cfg.Site.OriginAllowlist))
	if cfg.Site.BaseURL != "" {
		origins = append(origins, cfg.Site.BaseURL)
	}
	origins = append(origins, cfg.Site.OriginAllowlist...)

	sessCfg, store, err := provideSession(runtime, kvStore, storeFactory)
	if err != nil {
		return Security{}, nil, err
	}

	serverOrigin := cfg.Site.BaseURL
	csrf := cfg.Web.Secure.CSRF
	csrf.ServerOrigin = serverOrigin
	rateLimitKey, err := rateLimitKeyFunc(cfg.Web.Server.TrustedProxies)
	if err != nil {
		return Security{}, nil, fmt.Errorf("security: rate limit key: %w", err)
	}

	return Security{
		CSRF:         csrf,
		Headers:      cfg.Web.Secure.Headers,
		CSP:          cfg.Web.Secure.CSP,
		Origins:      origins,
		AllowedHosts: cfg.Site.AllowedHosts,
		ServerOrigin: serverOrigin,
		RateLimitKey: rateLimitKey,
		Session:      sessCfg,
	}, store, nil
}

func provideSession(runtime Runtime, kvStore kv.Store, storeFactory func() session.Store) (session.Config, session.Store, error) {
	cfg := session.StoreConfig{
		Type:       runtime.Config.Web.SessionStore,
		Env:        string(runtime.Config.Env),
		TestingEnv: string(config.Testing),
		Settings:   runtime.Config.Web.Session,
	}

	switch runtime.Config.Web.SessionStore {
	case session.StoreTypeCookie:
		codec, err := cookiecodec.New(cookiecodec.Config{
			Name:    runtime.Config.Web.Session.CookieName,
			Secrets: [][]byte{runtime.Config.Web.Session.CookieKey},
			Mode:    cookiecodec.AEAD,
		})
		if err != nil {
			return session.Config{}, nil, fmt.Errorf("session: cookie store: %w", err)
		}
		cfg.Codec = codec
	case session.StoreTypeKV:
		kvs, ok := kvStore.(session.KVStore)
		if !ok {
			return session.Config{}, nil, ErrKVSessionStoreUnsupported
		}
		cfg.KVStore = kvs
		cfg.KVPrefix = runtime.Config.Web.Session.KVPrefix
	case session.StoreTypeMemory:
		cfg.TestFactory = storeFactory
	}

	return session.NewStoreFromConfig(cfg)
}

func rateLimitKeyFunc(trustedProxies []string) (ratelimit.KeyFunc, error) {
	if len(trustedProxies) == 0 {
		return ratelimit.RemoteAddrKey, nil
	}
	fn, err := ratelimit.RealIPFromTrustedXFF(trustedProxies...)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTrustedProxyCIDRInvalid, err)
	}
	return fn, nil
}
