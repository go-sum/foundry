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

	if err := cfgpkg.ConnectWithRetry(ctx, "kv", runtime.Logger, 3, func() error {
		return store.Ping(ctx)
	}); err != nil {
		_ = store.Close() // secondary error during startup failure; primary error returned below
		return nil, errors.Join(ErrKVStoreUnavailable, fmt.Errorf("kv: ping: %w", err))
	}
	return store, nil
}

// needsKV reports whether the web application requires the shared KV
// dependency at startup. The current web runtime always depends on KV for auth
// token nonces, and may also use it for sessions and rate limiting.
func needsKV(cfg *config.Config) bool {
	_ = cfg
	return true
}
