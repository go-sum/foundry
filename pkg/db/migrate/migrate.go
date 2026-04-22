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

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("migrate up: set dialect: %w", err)
	}

	if err := goose.UpContext(ctx, db, dir); err != nil {
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

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("migrate up-to: set dialect: %w", err)
	}

	if err := goose.UpToContext(ctx, db, dir, version); err != nil {
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

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("migrate down: set dialect: %w", err)
	}

	if err := goose.DownContext(ctx, db, dir); err != nil {
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

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("migrate down-to: set dialect: %w", err)
	}

	if err := goose.DownToContext(ctx, db, dir, version); err != nil {
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

	if err := goose.SetDialect("postgres"); err != nil {
		return nil, fmt.Errorf("migrate status: set dialect: %w", err)
	}

	migrations, err := goose.CollectMigrations(dir, 0, goose.MaxVersion)
	if err != nil {
		return nil, fmt.Errorf("migrate status: collect: %w", err)
	}

	current, err := goose.GetDBVersionContext(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("migrate status: db version: %w", err)
	}

	var statuses []MigrationStatus
	for _, m := range migrations {
		s := MigrationStatus{
			Version: m.Version,
			Source:  m.Source,
			Applied: m.Version <= current,
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

	if err := goose.SetDialect("postgres"); err != nil {
		return 0, fmt.Errorf("migrate version: set dialect: %w", err)
	}

	v, err := goose.GetDBVersionContext(ctx, db)
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

	if err := goose.SetDialect("postgres"); err != nil {
		return "", fmt.Errorf("migrate create: set dialect: %w", err)
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

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("migrate: open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	return db, nil
}
