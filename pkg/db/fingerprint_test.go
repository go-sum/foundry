package db

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestStoreFingerprint_CreatesTableAndStoresValue(t *testing.T) {
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
	pool.Exec(ctx, "DROP TABLE IF EXISTS _schema_fingerprint") //nolint:errcheck
	t.Cleanup(func() {
		pool.Exec(ctx, "DROP TABLE IF EXISTS _schema_fingerprint") //nolint:errcheck
	})

	const fp = "abc123"
	if err := StoreFingerprint(ctx, pool, fp); err != nil {
		t.Fatalf("StoreFingerprint: %v", err)
	}

	var got string
	if err := pool.QueryRow(ctx, "SELECT fingerprint FROM _schema_fingerprint WHERE id = 1").Scan(&got); err != nil {
		t.Fatalf("SELECT after StoreFingerprint: %v", err)
	}
	if got != fp {
		t.Fatalf("stored fingerprint = %q, want %q", got, fp)
	}
}

func TestStoreFingerprint_UpdatesExistingValue(t *testing.T) {
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
	pool.Exec(ctx, "DROP TABLE IF EXISTS _schema_fingerprint") //nolint:errcheck
	t.Cleanup(func() {
		pool.Exec(ctx, "DROP TABLE IF EXISTS _schema_fingerprint") //nolint:errcheck
	})

	if err := StoreFingerprint(ctx, pool, "first"); err != nil {
		t.Fatalf("StoreFingerprint (first): %v", err)
	}
	if err := StoreFingerprint(ctx, pool, "second"); err != nil {
		t.Fatalf("StoreFingerprint (second): %v", err)
	}

	var got string
	if err := pool.QueryRow(ctx, "SELECT fingerprint FROM _schema_fingerprint WHERE id = 1").Scan(&got); err != nil {
		t.Fatalf("SELECT after second store: %v", err)
	}
	if got != "second" {
		t.Fatalf("fingerprint = %q, want %q", got, "second")
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
	pool.Exec(ctx, "DROP TABLE IF EXISTS _schema_fingerprint") //nolint:errcheck
	t.Cleanup(func() {
		pool.Exec(ctx, "DROP TABLE IF EXISTS _schema_fingerprint") //nolint:errcheck
	})

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
	pool.Exec(ctx, "DROP TABLE IF EXISTS _schema_fingerprint") //nolint:errcheck
	t.Cleanup(func() {
		pool.Exec(ctx, "DROP TABLE IF EXISTS _schema_fingerprint") //nolint:errcheck
	})

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
	pool.Exec(ctx, "DROP TABLE IF EXISTS _schema_fingerprint") //nolint:errcheck
	t.Cleanup(func() {
		pool.Exec(ctx, "DROP TABLE IF EXISTS _schema_fingerprint") //nolint:errcheck
	})

	err = VerifyFingerprint(ctx, pool, "anyvalue")
	if err == nil {
		t.Fatal("expected ErrFingerprintMissing, got nil")
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
	pool.Exec(ctx, "DROP TABLE IF EXISTS _schema_fingerprint") //nolint:errcheck
	t.Cleanup(func() {
		pool.Exec(ctx, "DROP TABLE IF EXISTS _schema_fingerprint") //nolint:errcheck
	})

	// Create the table via StoreFingerprint, then delete the single row.
	if err := StoreFingerprint(ctx, pool, "seed"); err != nil {
		t.Fatalf("StoreFingerprint (seed): %v", err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM _schema_fingerprint"); err != nil {
		t.Fatalf("DELETE FROM _schema_fingerprint: %v", err)
	}

	err = VerifyFingerprint(ctx, pool, "anyvalue")
	if err == nil {
		t.Fatal("expected ErrFingerprintMissing for empty table, got nil")
	}
	if !errors.Is(err, ErrFingerprintMissing) {
		t.Fatalf("errors.Is(err, ErrFingerprintMissing) = false; err = %v", err)
	}
}
