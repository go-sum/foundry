package app

import (
	"context"
	"fmt"

	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/web/ratelimit"

	config "github.com/go-sum/foundry/config"
)

func provideRateLimiter(_ context.Context, runtime Runtime, kvStore kv.Store) (*ratelimit.Limiter, error) {
	cfg := runtime.Config.RateLimit

	storeCfg := ratelimit.StoreConfig{
		Type:       cfg.Store.Type,
		Env:        string(runtime.Config.Env),
		TestingEnv: string(config.Testing),
		KVPrefix:   cfg.Store.Prefix,
	}
	if cfg.Store.Type == ratelimit.StoreTypeKV {
		backend, ok := kvStore.(ratelimit.KVBackend)
		if !ok {
			return nil, config.ErrRateLimitStoreUnsupported
		}
		storeCfg.KVBackend = backend
	}

	store, err := ratelimit.NewStoreFromConfig(storeCfg)
	if err != nil {
		return nil, fmt.Errorf("web/ratelimit store: %w", err)
	}

	limiter, err := ratelimit.New(ratelimit.Config{
		Store:    store,
		Profiles: cfg.Profiles(),
		Logger:   runtime.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("web/ratelimit: %w", err)
	}
	return limiter, nil
}
