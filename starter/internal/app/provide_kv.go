package app

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	cfgpkg "github.com/go-sum/foundry/pkg/config"
	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/kv/redisstore"

	config "github.com/go-sum/foundry/config"
)

func provideKVStore(ctx context.Context, runtime Runtime, factory func(context.Context, Runtime) (kv.Store, error)) (kv.Store, error) {
	if !needsKV(runtime.Config) {
		return nil, nil
	}

	var store kv.Store
	var err error
	if factory != nil {
		store, err = factory(ctx, runtime)
	} else {
		var tlsConfig *tls.Config
		if runtime.Config.KV.TLSEnabled {
			tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		store = redisstore.New(redisstore.Config{
			Addr:      runtime.Config.KV.Addr,
			Password:  runtime.Config.KV.Password,
			TLSConfig: tlsConfig,
		})
	}
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, fmt.Errorf("%w: no store returned", ErrKVStoreUnavailable)
	}

	attempts := 3
	if runtime.Config.Env == config.Testing {
		attempts = 1
	}
	if err := cfgpkg.ConnectWithRetry(ctx, "kv", runtime.Logger, attempts, func() error {
		return store.Ping(ctx)
	}); err != nil {
		_ = store.Close() // secondary error during startup failure; primary error returned below
		return nil, errors.Join(ErrKVStoreUnavailable, fmt.Errorf("kv: ping: %w", err))
	}
	return store, nil
}

// needsKV reports whether the application requires the shared KV dependency at
// startup. Outside testing, starter services such as auth nonce storage and
// the production rate-limit store depend on the configured KV service
// regardless of which session store implementation is selected. In testing,
// only the explicit kv session store path requires bringing up the shared KV
// dependency because rate limiting falls back to an in-memory store.
func needsKV(cfg *config.Config) bool {
	if cfg.Env != config.Testing {
		return true
	}
	return cfg.Web.SessionStore == "kv"
}
