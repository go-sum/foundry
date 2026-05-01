package app

import (
	"context"
	"errors"
	"testing"

	configpkg "github.com/go-sum/foundry/config"
	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/web/ratelimit"
)

type unsupportedKVStore struct{}

func (unsupportedKVStore) Ping(context.Context) error                  { return nil }
func (unsupportedKVStore) Get(context.Context, string) ([]byte, error) { return nil, kv.ErrNotFound }
func (unsupportedKVStore) Set(context.Context, string, []byte, kv.SetOptions) error {
	return nil
}
func (unsupportedKVStore) Delete(context.Context, ...string) error          { return nil }
func (unsupportedKVStore) Exists(context.Context, ...string) (int64, error) { return 0, nil }
func (unsupportedKVStore) Close() error                                     { return nil }

func testMemoryRateLimitsConfig() configpkg.RateLimitsConfig {
	cfg := configpkg.DefaultRateLimitsConfig()
	cfg.Store.Type = ratelimit.StoreTypeMemory
	return cfg
}

func TestProvideRateLimiter_Success(t *testing.T) {
	limiter, err := provideRateLimiter(context.Background(), Runtime{
		Config: &configpkg.Config{
			Env:       configpkg.Testing,
			RateLimit: testMemoryRateLimitsConfig(),
		},
	}, nil)
	if err != nil {
		t.Fatalf("provideRateLimiter() error = %v", err)
	}
	if limiter == nil {
		t.Fatal("provideRateLimiter() limiter = nil, want non-nil")
	}

	decision, err := limiter.Allow(context.Background(), string(configpkg.RateLimitRoutesAuth), "core")
	if err != nil {
		t.Fatalf("limiter.Allow() error = %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("limiter.Allow() Allowed = %v, want true", decision.Allowed)
	}
}

func TestProvideRateLimiter_UnsupportedKVStore_ReturnsError(t *testing.T) {
	_, err := provideRateLimiter(context.Background(), Runtime{
		Config: &configpkg.Config{
			Env:       configpkg.Production,
			RateLimit: configpkg.DefaultRateLimitsConfig(), // defaults to kv store type
		},
	}, unsupportedKVStore{})
	if err == nil {
		t.Fatal("provideRateLimiter() error = nil, want non-nil")
	}
	if !errors.Is(err, configpkg.ErrRateLimitStoreUnsupported) {
		t.Fatalf("error = %v, want ErrRateLimitStoreUnsupported", err)
	}
}
