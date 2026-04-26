package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const migrationsDDL = `CREATE TABLE IF NOT EXISTS _migrations (
    version     INTEGER     PRIMARY KEY,
    name        TEXT        NOT NULL,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fingerprint TEXT
)`

// Runner manages migration execution against a single database connection.
type Runner struct {
	db  *sql.DB
	dir string
}

// NewRunner opens a database/sql connection and returns a Runner.
// Does NOT call EnsureTable — caller must call it explicitly.
func NewRunner(dsn, dir string) (*Runner, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("migrate: open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	return &Runner{db: db, dir: dir}, nil
}

// Close closes the underlying DB connection.
func (r *Runner) Close() error {
	return r.db.Close()
}

// EnsureTable creates the _migrations table if it does not exist.
func (r *Runner) EnsureTable(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, migrationsDDL)
	if err != nil {
		return fmt.Errorf("migrate: ensure table: %w", err)
	}
	return nil
}

// Applied returns a set of version numbers already recorded in _migrations.
func (r *Runner) Applied(ctx context.Context) (map[int64]bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT version FROM _migrations`)
	if err != nil {
		return nil, fmt.Errorf("migrate: query applied: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	applied := make(map[int64]bool)
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("migrate: scan applied: %w", err)
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

// appliedMap returns a map of version → applied_at for all applied migrations.
func (r *Runner) appliedMap(ctx context.Context) (map[int64]time.Time, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT version, applied_at FROM _migrations`)
	if err != nil {
		return nil, fmt.Errorf("migrate: query applied map: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	m := make(map[int64]time.Time)
	for rows.Next() {
		var v int64
		var t time.Time
		if err := rows.Scan(&v, &t); err != nil {
			return nil, fmt.Errorf("migrate: scan applied map: %w", err)
		}
		m[v] = t
	}
	return m, rows.Err()
}

// Up applies all pending migrations. Each migration runs in its own transaction.
// Returns count of applied migrations.
func (r *Runner) Up(ctx context.Context) (int, error) {
	if err := r.EnsureTable(ctx); err != nil {
		return 0, err
	}

	applied, err := r.Applied(ctx)
	if err != nil {
		return 0, err
	}

	migrations, err := ParseDir(r.dir)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}
		if err := r.applyUp(ctx, m); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// UpTo applies pending migrations up to and including the given version.
func (r *Runner) UpTo(ctx context.Context, version int64) (int, error) {
	if err := r.EnsureTable(ctx); err != nil {
		return 0, err
	}

	applied, err := r.Applied(ctx)
	if err != nil {
		return 0, err
	}

	migrations, err := ParseDir(r.dir)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, m := range migrations {
		if m.Version > version {
			break
		}
		if applied[m.Version] {
			continue
		}
		if err := r.applyUp(ctx, m); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// Down rolls back the most recently applied migration.
func (r *Runner) Down(ctx context.Context) error {
	current, err := r.Version(ctx)
	if err != nil {
		return err
	}
	if current == 0 {
		return nil
	}

	migrations, err := ParseDir(r.dir)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if m.Version == current {
			return r.applyDown(ctx, m)
		}
	}
	return fmt.Errorf("migrate: no migration file found for version %d", current)
}

// DownTo rolls back to and including the given version (rolls back all versions >= version).
func (r *Runner) DownTo(ctx context.Context, version int64) error {
	applied, err := r.Applied(ctx)
	if err != nil {
		return err
	}

	migrations, err := ParseDir(r.dir)
	if err != nil {
		return err
	}

	// Roll back in reverse order.
	for i := len(migrations) - 1; i >= 0; i-- {
		m := migrations[i]
		if m.Version < version {
			break
		}
		if !applied[m.Version] {
			continue
		}
		if err := r.applyDown(ctx, m); err != nil {
			return err
		}
	}
	return nil
}

// Status returns state of all known migration files against applied set.
func (r *Runner) Status(ctx context.Context) ([]MigrationStatus, error) {
	migrations, err := ParseDir(r.dir)
	if err != nil {
		return nil, err
	}

	appliedMap, err := r.appliedMap(ctx)
	if err != nil {
		return nil, err
	}

	statuses := make([]MigrationStatus, len(migrations))
	for i, m := range migrations {
		t, ok := appliedMap[m.Version]
		statuses[i] = MigrationStatus{
			Version:   m.Version,
			Name:      m.Name,
			Applied:   ok,
			AppliedAt: t,
			Source:    m.Source,
		}
	}
	return statuses, nil
}

// Version returns the highest applied version (0 if none).
func (r *Runner) Version(ctx context.Context) (int64, error) {
	var v int64
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM _migrations`).Scan(&v)
	if err != nil {
		return 0, fmt.Errorf("migrate: get version: %w", err)
	}
	return v, nil
}

// StoreFingerprint clears all fingerprints and sets it on the max-version row.
func (r *Runner) StoreFingerprint(ctx context.Context, fingerprint string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE _migrations SET fingerprint = NULL WHERE fingerprint IS NOT NULL`)
	if err != nil {
		return fmt.Errorf("migrate: store fingerprint clear: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE _migrations SET fingerprint = $1 WHERE version = (SELECT MAX(version) FROM _migrations)`,
		fingerprint,
	)
	if err != nil {
		return fmt.Errorf("migrate: store fingerprint set: %w", err)
	}
	return nil
}

func (r *Runner) applyUp(ctx context.Context, m *Migration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrate: begin tx for v%d: %w", m.Version, err)
	}
	defer tx.Rollback() //nolint:errcheck

	for _, stmt := range SplitStatements(m.UpSQL) {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: apply v%d: %w", m.Version, err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO _migrations (version, name) VALUES ($1, $2)`,
		m.Version, m.Name,
	); err != nil {
		return fmt.Errorf("migrate: record v%d: %w", m.Version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrate: commit v%d: %w", m.Version, err)
	}
	return nil
}

func (r *Runner) applyDown(ctx context.Context, m *Migration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrate: begin tx for rollback v%d: %w", m.Version, err)
	}
	defer tx.Rollback() //nolint:errcheck

	for _, stmt := range SplitStatements(m.DownSQL) {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: rollback v%d: %w", m.Version, err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM _migrations WHERE version = $1`,
		m.Version,
	); err != nil {
		return fmt.Errorf("migrate: delete record v%d: %w", m.Version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrate: commit rollback v%d: %w", m.Version, err)
	}
	return nil
}
