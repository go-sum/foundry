package kv

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("kv: key not found")
	ErrClosed   = errors.New("kv: store closed")
	ErrConflict = errors.New("kv: version conflict")
)

// SetOptions configures a Set operation.
type SetOptions struct {
	TTL time.Duration // 0 means no expiry
}

// Store is the minimal key-value store contract.
type Store interface {
	Ping(ctx context.Context) error
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, opts SetOptions) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
	Close() error
}

// Scanner extends a Store with pattern-based key iteration.
type Scanner interface {
	Scan(ctx context.Context, pattern string, fn func(key string) error) error
}
