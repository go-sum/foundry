package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-sum/config"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Option configures pool creation.
type Option func(*pgxpool.Config)

// WithMaxConns sets the maximum number of pool connections.
func WithMaxConns(n int32) Option {
	return func(c *pgxpool.Config) {
		c.MaxConns = n
	}
}

// DSN resolves the database connection string from Docker secrets or environment.
func DSN() (string, error) {
	url := config.ExpandSecret("DATABASE_URL")
	if url == "" {
		return "", fmt.Errorf("DATABASE_URL is required (set as Docker secret or environment variable)")
	}
	return url, nil
}

// Connect resolves DSN automatically then creates a pgxpool.
func Connect(ctx context.Context, opts ...Option) (*pgxpool.Pool, error) {
	dsn, err := DSN()
	if err != nil {
		return nil, err
	}
	return ConnectDSN(ctx, dsn, opts...)
}

// ConnectDSN creates a pgxpool from the given DSN and verifies connectivity.
func ConnectDSN(ctx context.Context, dsn string, opts ...Option) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("db: parse config: %w", err)
	}

	for _, o := range opts {
		o(cfg)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}

	return pool, nil
}

// Health pings the pool and verifies that each named table exists in the public schema.
func Health(ctx context.Context, pool *pgxpool.Pool, tables ...string) error {
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("db health: ping: %w", err)
	}

	for _, table := range tables {
		var exists bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)`, table).Scan(&exists)
		if err != nil {
			return fmt.Errorf("db health: check table %q: %w", table, err)
		}
		if !exists {
			return fmt.Errorf("db health: table %q not found", table)
		}
	}

	return nil
}

// IsUniqueViolation reports whether err is a PostgreSQL unique-constraint violation (23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
