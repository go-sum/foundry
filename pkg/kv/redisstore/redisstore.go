package redisstore

import (
	"cmp"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/go-sum/foundry/pkg/kv"
	"github.com/redis/go-redis/v9"
)

// Config holds connection parameters for a Redis-protocol server.
type Config struct {
	Addr         string        `validate:"required"      help:"host:port of the Redis-protocol server"`
	Password     string        `                         help:"authentication password (empty for no auth)"`
	DB           int           `validate:"min=0,max=15"  help:"database number (default 0)"`
	PoolSize     int           `validate:"min=0"         help:"maximum connections (0 uses default of 10)"`
	MinIdleConns int           `validate:"min=0"         help:"minimum idle connections"`
	DialTimeout  time.Duration `                         help:"connection dial timeout (default 5s)"`
	ReadTimeout  time.Duration `                         help:"read timeout (default 3s)"`
	WriteTimeout time.Duration `                         help:"write timeout (default 3s)"`
	TLSConfig    *tls.Config   `                         help:"optional TLS config for managed Redis services"`
}

// RedisStore implements kv.Store and kv.Scanner backed by a Redis-protocol server.
type RedisStore struct {
	client *redis.Client
}

var (
	_ kv.Store   = (*RedisStore)(nil)
	_ kv.Scanner = (*RedisStore)(nil)
)

// New creates a RedisStore from cfg. It does not dial or ping; call Ping to
// verify connectivity before use.
func New(cfg Config) *RedisStore {
	return &RedisStore{
		client: redis.NewClient(&redis.Options{
			Addr:         cfg.Addr,
			Password:     cfg.Password,
			DB:           cfg.DB,
			PoolSize:     cmp.Or(cfg.PoolSize, 10),
			MinIdleConns: cfg.MinIdleConns,
			DialTimeout:  cmp.Or(cfg.DialTimeout, 5*time.Second),
			ReadTimeout:  cmp.Or(cfg.ReadTimeout, 3*time.Second),
			WriteTimeout: cmp.Or(cfg.WriteTimeout, 3*time.Second),
			TLSConfig:    cfg.TLSConfig,
		}),
	}
}

func (s *RedisStore) Ping(ctx context.Context) error {
	if err := s.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("kv: ping: %w", err)
	}
	return nil
}

func (s *RedisStore) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, kv.ErrNotFound
		}
		return nil, fmt.Errorf("kv: get %q: %w", key, err)
	}
	return val, nil
}

func (s *RedisStore) Set(ctx context.Context, key string, value []byte, opts kv.SetOptions) error {
	if err := s.client.Set(ctx, key, value, opts.TTL).Err(); err != nil {
		return fmt.Errorf("kv: set %q: %w", key, err)
	}
	return nil
}

func (s *RedisStore) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := s.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("kv: delete: %w", err)
	}
	return nil
}

func (s *RedisStore) Exists(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	n, err := s.client.Exists(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("kv: exists: %w", err)
	}
	return n, nil
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

func (s *RedisStore) Scan(ctx context.Context, pattern string, fn func(key string) error) error {
	var cursor uint64
	for {
		keys, next, err := s.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("kv: scan %q: %w", pattern, err)
		}
		for _, key := range keys {
			if err := fn(key); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return nil
}
