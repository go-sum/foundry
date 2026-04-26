package db

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestStoreFingerprint_SetsValueOnLatestRow(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := ConnectDSN(context.Background(), dsn)
	if err != nil {
		t.Fatalf("ConnectDSN: %v", err)
	}
	t.Cleanup(pool.Close)

	ctx := context.Background()
	pool.Exec(ctx, "DROP TABLE IF EXISTS _migrations") //nolint:errcheck
	t.Cleanup(func() {
		pool.Exec(ctx, "DROP TABLE IF EXISTS _migrations") //nolint:errcheck
	})

	// Create _migrations table and insert a row.
	_, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS _migrations (
		version     INTEGER     PRIMARY KEY,
		name        TEXT        NOT NULL,
		applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		fingerprint TEXT
	)`)
	if err != nil {
		t.Fatalf("create _migrations: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO _migrations (version, name) VALUES (1, 'initial')`); err != nil {
		t.Fatalf("insert migration row: %v", err)
	}

	const fp = "abc123"
	if err := StoreFingerprint(ctx, pool, fp); err != nil {
		t.Fatalf("StoreFingerprint: %v", err)
	}

	var got *string
	if err := pool.QueryRow(ctx, `SELECT fingerprint FROM _migrations WHERE version = 1`).Scan(&got); err != nil {
		t.Fatalf("SELECT after StoreFingerprint: %v", err)
	}
	if got == nil || *got != fp {
		t.Fatalf("stored fingerprint = %v, want %q", got, fp)
	}
}

func TestStoreFingerprint_NoTable_ReturnsMissing(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := ConnectDSN(context.Background(), dsn)
	if err != nil {
		t.Fatalf("ConnectDSN: %v", err)
	}
	t.Cleanup(pool.Close)

	ctx := context.Background()
	pool.Exec(ctx, "DROP TABLE IF EXISTS _migrations") //nolint:errcheck
	t.Cleanup(func() {
		pool.Exec(ctx, "DROP TABLE IF EXISTS _migrations") //nolint:errcheck
	})

	err = StoreFingerprint(ctx, pool, "fp1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrFingerprintMissing) {
		t.Fatalf("errors.Is(err, ErrFingerprintMissing) = false; err = %v", err)
	}
}

func TestVerifyFingerprint_Match_ReturnsNil(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := ConnectDSN(context.Background(), dsn)
	if err != nil {
		t.Fatalf("ConnectDSN: %v", err)
	}
	t.Cleanup(pool.Close)

	ctx := context.Background()
	pool.Exec(ctx, "DROP TABLE IF EXISTS _migrations") //nolint:errcheck
	t.Cleanup(func() {
		pool.Exec(ctx, "DROP TABLE IF EXISTS _migrations") //nolint:errcheck
	})

	_, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS _migrations (
		version     INTEGER     PRIMARY KEY,
		name        TEXT        NOT NULL,
		applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		fingerprint TEXT
	)`)
	if err != nil {
		t.Fatalf("create _migrations: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO _migrations (version, name) VALUES (1, 'initial')`); err != nil {
		t.Fatalf("insert migration row: %v", err)
	}

	const fp = "fp1"
	if err := StoreFingerprint(ctx, pool, fp); err != nil {
		t.Fatalf("StoreFingerprint: %v", err)
	}

	if err := VerifyFingerprint(ctx, pool, fp); err != nil {
		t.Fatalf("VerifyFingerprint with matching fp: %v", err)
	}
}

func TestVerifyFingerprint_Mismatch_ReturnsMismatchError(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := ConnectDSN(context.Background(), dsn)
	if err != nil {
		t.Fatalf("ConnectDSN: %v", err)
	}
	t.Cleanup(pool.Close)

	ctx := context.Background()
	pool.Exec(ctx, "DROP TABLE IF EXISTS _migrations") //nolint:errcheck
	t.Cleanup(func() {
		pool.Exec(ctx, "DROP TABLE IF EXISTS _migrations") //nolint:errcheck
	})

	_, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS _migrations (
		version     INTEGER     PRIMARY KEY,
		name        TEXT        NOT NULL,
		applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		fingerprint TEXT
	)`)
	if err != nil {
		t.Fatalf("create _migrations: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO _migrations (version, name) VALUES (1, 'initial')`); err != nil {
		t.Fatalf("insert migration row: %v", err)
	}

	if err := StoreFingerprint(ctx, pool, "fp1"); err != nil {
		t.Fatalf("StoreFingerprint: %v", err)
	}

	err = VerifyFingerprint(ctx, pool, "fp2")
	if err == nil {
		t.Fatal("expected ErrFingerprintMismatch, got nil")
	}
	if !errors.Is(err, ErrFingerprintMismatch) {
		t.Fatalf("errors.Is(err, ErrFingerprintMismatch) = false; err = %v", err)
	}
}

func TestVerifyFingerprint_NoTable_ReturnsMissingError(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := ConnectDSN(context.Background(), dsn)
	if err != nil {
		t.Fatalf("ConnectDSN: %v", err)
	}
	t.Cleanup(pool.Close)

	ctx := context.Background()
	pool.Exec(ctx, "DROP TABLE IF EXISTS _migrations") //nolint:errcheck
	t.Cleanup(func() {
		pool.Exec(ctx, "DROP TABLE IF EXISTS _migrations") //nolint:errcheck
	})

	err = VerifyFingerprint(ctx, pool, "anyvalue")
	if err == nil {
		t.Fatal("expected ErrFingerprintMissing, got nil")
	}
	if !errors.Is(err, ErrFingerprintMissing) {
		t.Fatalf("errors.Is(err, ErrFingerprintMissing) = false; err = %v", err)
	}
}

func TestVerifyFingerprint_NullFingerprint_ReturnsMissingError(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := ConnectDSN(context.Background(), dsn)
	if err != nil {
		t.Fatalf("ConnectDSN: %v", err)
	}
	t.Cleanup(pool.Close)

	ctx := context.Background()
	pool.Exec(ctx, "DROP TABLE IF EXISTS _migrations") //nolint:errcheck
	t.Cleanup(func() {
		pool.Exec(ctx, "DROP TABLE IF EXISTS _migrations") //nolint:errcheck
	})

	_, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS _migrations (
		version     INTEGER     PRIMARY KEY,
		name        TEXT        NOT NULL,
		applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		fingerprint TEXT
	)`)
	if err != nil {
		t.Fatalf("create _migrations: %v", err)
	}
	// Insert row without fingerprint (NULL).
	if _, err := pool.Exec(ctx, `INSERT INTO _migrations (version, name) VALUES (1, 'initial')`); err != nil {
		t.Fatalf("insert migration row: %v", err)
	}

	err = VerifyFingerprint(ctx, pool, "anyvalue")
	if err == nil {
		t.Fatal("expected ErrFingerprintMissing for null fingerprint, got nil")
	}
	if !errors.Is(err, ErrFingerprintMissing) {
		t.Fatalf("errors.Is(err, ErrFingerprintMissing) = false; err = %v", err)
	}
}

func TestVerifyFingerprint_EmptyTable_ReturnsMissingError(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := ConnectDSN(context.Background(), dsn)
	if err != nil {
		t.Fatalf("ConnectDSN: %v", err)
	}
	t.Cleanup(pool.Close)

	ctx := context.Background()
	pool.Exec(ctx, "DROP TABLE IF EXISTS _migrations") //nolint:errcheck
	t.Cleanup(func() {
		pool.Exec(ctx, "DROP TABLE IF EXISTS _migrations") //nolint:errcheck
	})

	_, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS _migrations (
		version     INTEGER     PRIMARY KEY,
		name        TEXT        NOT NULL,
		applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		fingerprint TEXT
	)`)
	if err != nil {
		t.Fatalf("create _migrations: %v", err)
	}

	err = VerifyFingerprint(ctx, pool, "anyvalue")
	if err == nil {
		t.Fatal("expected ErrFingerprintMissing for empty table, got nil")
	}
	if !errors.Is(err, ErrFingerprintMissing) {
		t.Fatalf("errors.Is(err, ErrFingerprintMissing) = false; err = %v", err)
	}
}
