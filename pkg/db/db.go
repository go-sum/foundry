package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-sum/config"
	"github.com/jackc/pgx/v5"
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

// WithMinConns sets the minimum number of pool connections.
func WithMinConns(n int32) Option {
	return func(c *pgxpool.Config) {
		c.MinConns = n
	}
}

// WithMaxConnLifetime sets the maximum lifetime of a pool connection.
func WithMaxConnLifetime(d time.Duration) Option {
	return func(c *pgxpool.Config) {
		c.MaxConnLifetime = d
	}
}

// WithMaxConnIdleTime sets the maximum idle time of a pool connection.
func WithMaxConnIdleTime(d time.Duration) Option {
	return func(c *pgxpool.Config) {
		c.MaxConnIdleTime = d
	}
}

// WithHealthCheckPeriod sets how often the pool checks idle connections.
func WithHealthCheckPeriod(d time.Duration) Option {
	return func(c *pgxpool.Config) {
		c.HealthCheckPeriod = d
	}
}

// WithConnectTimeout sets the per-connection dial timeout.
func WithConnectTimeout(d time.Duration) Option {
	return func(c *pgxpool.Config) {
		c.ConnConfig.ConnectTimeout = d
	}
}

// WithStatementTimeout installs an AfterConnect hook that runs
// SET statement_timeout = <ms> on every new connection, chaining with any
// existing AfterConnect hook already present in the config.
func WithStatementTimeout(d time.Duration) Option {
	return func(c *pgxpool.Config) {
		ms := d.Milliseconds()
		prev := c.AfterConnect
		c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			if prev != nil {
				if err := prev(ctx, conn); err != nil {
					return err
				}
			}
			_, err := conn.Exec(ctx, fmt.Sprintf("SET statement_timeout = %d", ms))
			return err
		}
	}
}

// WithSlowQueryLogger sets a tracer on the connection config that logs queries
// exceeding threshold at Warn level using logger.
func WithSlowQueryLogger(logger *slog.Logger, threshold time.Duration) Option {
	return func(c *pgxpool.Config) {
		c.ConnConfig.Tracer = &slowQueryTracer{
			logger:    logger,
			threshold: threshold,
		}
	}
}

// WithProductionDefaults applies sensible production pool defaults:
// min 2 / max 20 connections, 30-minute lifetime, 5-minute idle,
// 30-second health check, 5-second connect timeout, 30-second statement timeout.
func WithProductionDefaults() Option {
	return func(c *pgxpool.Config) {
		WithMinConns(2)(c)
		WithMaxConns(20)(c)
		WithMaxConnLifetime(30 * time.Minute)(c)
		WithMaxConnIdleTime(5 * time.Minute)(c)
		WithHealthCheckPeriod(30 * time.Second)(c)
		WithConnectTimeout(5 * time.Second)(c)
		WithStatementTimeout(30 * time.Second)(c)
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

// Health pings the pool and verifies that each named table is queryable.
// A 5-second timeout is applied to the entire check.
func Health(ctx context.Context, pool *pgxpool.Pool, tables ...string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("db health: ping: %w", err)
	}

	for _, table := range tables {
		q := fmt.Sprintf("SELECT 1 FROM %s LIMIT 0", pgx.Identifier{table}.Sanitize())
		if _, err := pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("db health: table %q: %w", table, err)
		}
	}

	return nil
}

// LogPoolStats starts a goroutine that logs pool statistics at every interval
// until ctx is cancelled.
func LogPoolStats(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s := pool.Stat()
				logger.InfoContext(ctx, "db pool stats",
					"total_conns", s.TotalConns(),
					"idle_conns", s.IdleConns(),
					"acquired_conns", s.AcquiredConns(),
					"acquire_count", s.AcquireCount(),
					"acquire_duration", s.AcquireDuration(),
				)
			}
		}
	}()
}

// WithTx runs fn inside a transaction. It commits on nil return and rolls back
// on error. A deferred rollback is always registered; rollback after commit is
// a no-op in pgx.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("db: commit tx: %w", err)
	}

	return nil
}

// isPgCode reports whether err is a pgconn.PgError with the given SQLSTATE code.
func isPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

// IsUniqueViolation reports whether err is a PostgreSQL unique-constraint violation (23505).
func IsUniqueViolation(err error) bool {
	return isPgCode(err, "23505")
}

// IsForeignKeyViolation reports whether err is a PostgreSQL foreign-key violation (23503).
func IsForeignKeyViolation(err error) bool {
	return isPgCode(err, "23503")
}

// IsDeadlock reports whether err is a PostgreSQL deadlock detected error (40P01).
func IsDeadlock(err error) bool {
	return isPgCode(err, "40P01")
}

// IsSerializationFailure reports whether err is a PostgreSQL serialization failure (40001).
func IsSerializationFailure(err error) bool {
	return isPgCode(err, "40001")
}

// slowQueryTracer implements pgx.QueryTracer and logs queries that exceed threshold.
type slowQueryTracer struct {
	logger    *slog.Logger
	threshold time.Duration
}

// queryTrace holds the data stored in context between TraceQueryStart and TraceQueryEnd.
type queryTrace struct {
	sql   string
	start time.Time
}

// ctxKey is an unexported type for context keys in this package.
type ctxKey struct{}

// TraceQueryStart stores the query start time and SQL in the returned context.
func (t *slowQueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, ctxKey{}, queryTrace{sql: data.SQL, start: time.Now()})
}

// TraceQueryEnd reads the start time and logs a warning if the duration exceeds threshold.
func (t *slowQueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
	qt, ok := ctx.Value(ctxKey{}).(queryTrace)
	if !ok {
		return
	}
	d := time.Since(qt.start)
	if d >= t.threshold {
		t.logger.WarnContext(ctx, "slow query",
			"sql", qt.sql,
			"duration", d,
		)
	}
}
