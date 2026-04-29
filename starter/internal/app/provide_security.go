package app

import (
	"context"
	"encoding/hex"
	"fmt"

	cfgpkg "github.com/go-sum/foundry/pkg/config"
	"github.com/go-sum/foundry/pkg/web/cookiecodec"
	"github.com/go-sum/foundry/pkg/web/session"

	config "github.com/go-sum/foundry/config"
)

func provideSecurity(_ context.Context, runtime Runtime, storeFactory func() session.Store) (Security, session.Store, error) {
	cfg := runtime.Config

	origins := make([]string, 0, 1+len(cfg.Site.OriginAllowlist))
	if cfg.Site.BaseURL != "" {
		origins = append(origins, cfg.Site.BaseURL)
	}
	origins = append(origins, cfg.Site.OriginAllowlist...)

	sessCfg, store, err := provideSession(runtime, storeFactory)
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

func provideSession(runtime Runtime, storeFactory func() session.Store) (session.Config, session.Store, error) {
	var store session.Store
	switch runtime.Config.SessionStore {
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
	default:
		if storeFactory != nil {
			store = storeFactory()
		} else {
			store = session.NewMemoryStore()
		}
	}
	return session.NewConfig(runtime.Config.Session, store), store, nil
}

