package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// MigrationStatus describes the state of a single migration file.
type MigrationStatus struct {
	Version   int64
	Applied   bool
	AppliedAt time.Time
	Source    string
}

// Up applies all pending migrations from dir to the database at dsn.
func Up(ctx context.Context, dsn, dir string) (err error) {
	db, err := openDB(dsn)
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck

	p, err := newProvider(db, dir)
	if err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	defer p.Close() //nolint:errcheck

	if _, err := p.Up(ctx); err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}

// UpTo applies migrations up to and including the given version.
func UpTo(ctx context.Context, dsn, dir string, version int64) (err error) {
	db, err := openDB(dsn)
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck

	p, err := newProvider(db, dir)
	if err != nil {
		return fmt.Errorf("migrate up-to: %w", err)
	}
	defer p.Close() //nolint:errcheck

	if _, err := p.UpTo(ctx, version); err != nil {
		return fmt.Errorf("migrate up-to: %w", err)
	}

	return nil
}

// Down rolls back the most recently applied migration.
func Down(ctx context.Context, dsn, dir string) (err error) {
	db, err := openDB(dsn)
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck

	p, err := newProvider(db, dir)
	if err != nil {
		return fmt.Errorf("migrate down: %w", err)
	}
	defer p.Close() //nolint:errcheck

	if _, err := p.Down(ctx); err != nil {
		return fmt.Errorf("migrate down: %w", err)
	}

	return nil
}

// DownTo rolls back migrations to and including the given version.
func DownTo(ctx context.Context, dsn, dir string, version int64) (err error) {
	db, err := openDB(dsn)
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck

	p, err := newProvider(db, dir)
	if err != nil {
		return fmt.Errorf("migrate down-to: %w", err)
	}
	defer p.Close() //nolint:errcheck

	if _, err := p.DownTo(ctx, version); err != nil {
		return fmt.Errorf("migrate down-to: %w", err)
	}

	return nil
}

// Status returns the state of each known migration.
func Status(ctx context.Context, dsn, dir string) (_ []MigrationStatus, err error) {
	db, err := openDB(dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close() //nolint:errcheck

	p, err := newProvider(db, dir)
	if err != nil {
		return nil, fmt.Errorf("migrate status: %w", err)
	}
	defer p.Close() //nolint:errcheck

	results, err := p.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrate status: %w", err)
	}

	var statuses []MigrationStatus
	for _, r := range results {
		s := MigrationStatus{
			Version:   r.Source.Version,
			Applied:   r.State == goose.StateApplied,
			AppliedAt: r.AppliedAt,
			Source:    r.Source.Path,
		}
		statuses = append(statuses, s)
	}

	return statuses, nil
}

// Version returns the current applied migration version (0 if none applied).
func Version(ctx context.Context, dsn string) (_ int64, err error) {
	db, err := openDB(dsn)
	if err != nil {
		return 0, err
	}
	defer db.Close() //nolint:errcheck

	p, err := newProvider(db, os.TempDir())
	if err != nil {
		return 0, fmt.Errorf("migrate version: %w", err)
	}
	defer p.Close() //nolint:errcheck

	v, err := p.GetDBVersion(ctx)
	if err != nil {
		return 0, fmt.Errorf("migrate version: %w", err)
	}
	return v, nil
}

// Create generates a new empty SQL migration file in dir with the given name.
// It returns the path of the created file.
func Create(dir, name string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("migrate create: mkdir: %w", err)
	}

	if err := goose.Create(nil, dir, name, "sql"); err != nil {
		return "", fmt.Errorf("migrate create: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("migrate create: read dir: %w", err)
	}

	var lastSQL string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			lastSQL = filepath.Join(dir, e.Name())
		}
	}

	return lastSQL, nil
}

func newProvider(db *sql.DB, dir string) (*goose.Provider, error) {
	return goose.NewProvider(goose.DialectPostgres, db, os.DirFS(dir))
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("migrate: open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	return db, nil
}
