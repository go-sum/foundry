package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrFingerprintMismatch is returned when the stored schema fingerprint differs
// from the fingerprint computed from the binary's embedded schema.
var ErrFingerprintMismatch = errors.New("db: schema fingerprint mismatch")

// ErrFingerprintMissing is returned when no fingerprint has been stored yet
// (first deploy or upgrade from a pre-fingerprint version).
var ErrFingerprintMissing = errors.New("db: schema fingerprint not stored")

// StoreFingerprint sets the fingerprint on the latest _migrations row.
// Clears fingerprint from all other rows first.
func StoreFingerprint(ctx context.Context, pool *pgxpool.Pool, fingerprint string) error {
	_, err := pool.Exec(ctx, `UPDATE _migrations SET fingerprint = NULL WHERE fingerprint IS NOT NULL`)
	if err != nil {
		if isPgCode(err, "42P01") {
			return ErrFingerprintMissing
		}
		return fmt.Errorf("db: store fingerprint: %w", err)
	}
	_, err = pool.Exec(ctx,
		`UPDATE _migrations SET fingerprint = $1 WHERE version = (SELECT MAX(version) FROM _migrations)`,
		fingerprint,
	)
	if err != nil {
		return fmt.Errorf("db: store fingerprint: %w", err)
	}
	return nil
}

// VerifyFingerprint checks the fingerprint on the latest _migrations row.
func VerifyFingerprint(ctx context.Context, pool *pgxpool.Pool, fingerprint string) error {
	var stored *string
	err := pool.QueryRow(ctx,
		`SELECT fingerprint FROM _migrations ORDER BY version DESC LIMIT 1`,
	).Scan(&stored)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isPgCode(err, "42P01") {
			return ErrFingerprintMissing
		}
		return fmt.Errorf("db: verify fingerprint: %w", err)
	}
	if stored == nil || *stored == "" {
		return ErrFingerprintMissing
	}
	if *stored != fingerprint {
		return fmt.Errorf("%w: want %s, have %s", ErrFingerprintMismatch, fingerprint, *stored)
	}
	return nil
}
