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

const fingerprintDDL = `CREATE TABLE IF NOT EXISTS _schema_fingerprint (
    id          INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    fingerprint TEXT    NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`

// StoreFingerprint creates the fingerprint table if needed and upserts the
// fingerprint value.
func StoreFingerprint(ctx context.Context, pool *pgxpool.Pool, fingerprint string) error {
	if _, err := pool.Exec(ctx, fingerprintDDL); err != nil {
		return fmt.Errorf("db: store fingerprint: create table: %w", err)
	}
	const upsert = `INSERT INTO _schema_fingerprint (id, fingerprint, updated_at)
        VALUES (1, $1, NOW())
        ON CONFLICT (id) DO UPDATE SET fingerprint = $1, updated_at = NOW()`
	if _, err := pool.Exec(ctx, upsert, fingerprint); err != nil {
		return fmt.Errorf("db: store fingerprint: %w", err)
	}
	return nil
}

// VerifyFingerprint checks that the stored fingerprint matches the provided
// value. Returns ErrFingerprintMissing if the table or row does not exist.
// Returns ErrFingerprintMismatch if the stored value differs.
func VerifyFingerprint(ctx context.Context, pool *pgxpool.Pool, fingerprint string) error {
	var stored string
	err := pool.QueryRow(ctx, `SELECT fingerprint FROM _schema_fingerprint WHERE id = 1`).Scan(&stored)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isPgCode(err, "42P01") {
			return ErrFingerprintMissing
		}
		return fmt.Errorf("db: verify fingerprint: %w", err)
	}
	if stored != fingerprint {
		return fmt.Errorf("%w: want %s, have %s", ErrFingerprintMismatch, fingerprint, stored)
	}
	return nil
}
