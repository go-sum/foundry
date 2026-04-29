package app

import (
	"context"
	"encoding/hex"
	"fmt"

	cfgpkg "github.com/go-sum/foundry/pkg/config"
	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/web/cookiecodec"
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
	csrf := cfg.CSRF
	csrf.ServerOrigin = serverOrigin

	return Security{
		CSRF:         csrf,
		Headers:      cfg.Headers,
		CSP:          cfg.CSP,
		Origins:      origins,
		AllowedHosts: cfg.Site.AllowedHosts,
		ServerOrigin: serverOrigin,
		Session:      sessCfg,
	}, store, nil
}

func provideSession(runtime Runtime, kvStore kv.Store, storeFactory func() session.Store) (session.Config, session.Store, error) {
	var store session.Store
	sessionStore := runtime.Config.SessionStore

	switch sessionStore {
	case "cookie":
		keyHex := cfgpkg.ExpandSecret("SECURITY_SESSION_KEY")
		if keyHex == "" {
			return session.Config{}, nil, config.ErrSessionKeyMissing
		}
		key, err := hex.DecodeString(keyHex)
		if err != nil {
			return session.Config{}, nil, fmt.Errorf("%w: %w", config.ErrSessionKeyInvalid, err)
		}
		codec, err := cookiecodec.New(cookiecodec.Config{
			Name:    runtime.Config.Session.CookieName,
			Secrets: [][]byte{key},
			Mode:    cookiecodec.AEAD,
		})
		if err != nil {
			return session.Config{}, nil, fmt.Errorf("session: cookie store: %w", err)
		}
		store = session.NewCookieStore(codec)
	case "kv":
		backend, ok := kvStore.(session.KVBackend)
		if !ok {
			return session.Config{}, nil, config.ErrKVSessionStoreUnsupported
		}
		store = session.NewKVStore(backend, session.KVStoreConfig{
			Prefix: runtime.Config.Session.KVPrefix,
		})
	case "memory":
		if runtime.Config.Env != config.Testing {
			return session.Config{}, nil, config.ErrSessionStoreMemoryTestingOnly
		}
		if storeFactory != nil {
			store = storeFactory()
		} else {
			store = session.NewMemoryStore()
		}
	default:
		return session.Config{}, nil, fmt.Errorf("session: unsupported store %q", sessionStore)
	}
	return session.NewConfig(runtime.Config.Session, store), store, nil
}
