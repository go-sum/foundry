package migrate

import (
	"context"
	"fmt"
)

// Up applies all pending migrations from dir to the database at dsn.
func Up(ctx context.Context, dsn, dir string) error {
	r, err := NewRunner(dsn, dir)
	if err != nil {
		return err
	}
	defer r.Close() //nolint:errcheck

	if _, err := r.Up(ctx); err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// UpTo applies migrations up to and including the given version.
func UpTo(ctx context.Context, dsn, dir string, version int64) error {
	r, err := NewRunner(dsn, dir)
	if err != nil {
		return err
	}
	defer r.Close() //nolint:errcheck

	if _, err := r.UpTo(ctx, version); err != nil {
		return fmt.Errorf("migrate up-to: %w", err)
	}
	return nil
}

// Down rolls back the most recently applied migration.
func Down(ctx context.Context, dsn, dir string) error {
	r, err := NewRunner(dsn, dir)
	if err != nil {
		return err
	}
	defer r.Close() //nolint:errcheck

	if err := r.Down(ctx); err != nil {
		return fmt.Errorf("migrate down: %w", err)
	}
	return nil
}

// DownTo rolls back down to and including the given version.
func DownTo(ctx context.Context, dsn, dir string, version int64) error {
	r, err := NewRunner(dsn, dir)
	if err != nil {
		return err
	}
	defer r.Close() //nolint:errcheck

	if err := r.DownTo(ctx, version); err != nil {
		return fmt.Errorf("migrate down-to: %w", err)
	}
	return nil
}

// Status returns the state of each known migration file.
func Status(ctx context.Context, dsn, dir string) ([]MigrationStatus, error) {
	r, err := NewRunner(dsn, dir)
	if err != nil {
		return nil, err
	}
	defer r.Close() //nolint:errcheck

	statuses, err := r.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrate status: %w", err)
	}
	return statuses, nil
}

// Version returns the current applied migration version (0 if none applied).
func Version(ctx context.Context, dsn string) (int64, error) {
	r, err := NewRunner(dsn, "")
	if err != nil {
		return 0, err
	}
	defer r.Close() //nolint:errcheck

	v, err := r.Version(ctx)
	if err != nil {
		return 0, fmt.Errorf("migrate version: %w", err)
	}
	return v, nil
}

// StoreFingerprintDSN opens a connection and stores the fingerprint on the latest migration row.
func StoreFingerprintDSN(ctx context.Context, dsn, dir, fingerprint string) error {
	r, err := NewRunner(dsn, dir)
	if err != nil {
		return err
	}
	defer r.Close() //nolint:errcheck

	if err := r.StoreFingerprint(ctx, fingerprint); err != nil {
		return fmt.Errorf("migrate store fingerprint: %w", err)
	}
	return nil
}
