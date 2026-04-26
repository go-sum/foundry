package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// All tests in this file require a live PostgreSQL database.
// Set TEST_DATABASE_URL to run them; otherwise they are skipped.
// Example: TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/test_db

func requireDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	return dsn
}

// writeSQLMigration writes a numbered migration file with Up and Down sections.
func writeSQLMigration(t *testing.T, dir string, seq int, name, upSQL, downSQL string) string {
	t.Helper()
	fileName := fmt.Sprintf("%05d_%s.sql", seq, name)
	content := fmt.Sprintf("-- +migrate Up\n%s\n\n-- +migrate Down\n%s\n", upSQL, downSQL)
	path := filepath.Join(dir, fileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeSQLMigration %q: %v", fileName, err)
	}
	return path
}

// cleanupMigrationTable drops the _migrations table to give each test a clean slate.
func cleanupMigrationTable(t *testing.T, r *Runner) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = r.db.ExecContext(ctx, `DROP TABLE IF EXISTS _migrations`)
	})
}

// ---- Up / Version -----------------------------------------------------------

func TestRunner_Up_AppliesMigrations(t *testing.T) {
	dsn := requireDSN(t)
	dir := t.TempDir()

	writeSQLMigration(t, dir, 1, "create_alpha",
		"CREATE TABLE IF NOT EXISTS _test_alpha (id INT);",
		"DROP TABLE IF EXISTS _test_alpha;",
	)
	writeSQLMigration(t, dir, 2, "create_beta",
		"CREATE TABLE IF NOT EXISTS _test_beta (id INT);",
		"DROP TABLE IF EXISTS _test_beta;",
	)

	r, err := NewRunner(dsn, dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	defer r.Close() //nolint:errcheck
	cleanupMigrationTable(t, r)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = r.db.ExecContext(ctx, `DROP TABLE IF EXISTS _test_alpha`)
		_, _ = r.db.ExecContext(ctx, `DROP TABLE IF EXISTS _test_beta`)
	})

	ctx := context.Background()
	count, err := r.Up(ctx)
	if err != nil {
		t.Fatalf("Up: %v", err)
	}
	if count != 2 {
		t.Errorf("Up returned %d, want 2", count)
	}
}

func TestRunner_Up_Idempotent(t *testing.T) {
	dsn := requireDSN(t)
	dir := t.TempDir()

	writeSQLMigration(t, dir, 1, "create_idempotent",
		"CREATE TABLE IF NOT EXISTS _test_idem (id INT);",
		"DROP TABLE IF EXISTS _test_idem;",
	)

	r, err := NewRunner(dsn, dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	defer r.Close() //nolint:errcheck
	cleanupMigrationTable(t, r)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = r.db.ExecContext(ctx, `DROP TABLE IF EXISTS _test_idem`)
	})

	ctx := context.Background()
	// First run.
	if _, err := r.Up(ctx); err != nil {
		t.Fatalf("first Up: %v", err)
	}
	// Second run — nothing new to apply.
	count, err := r.Up(ctx)
	if err != nil {
		t.Fatalf("second Up: %v", err)
	}
	if count != 0 {
		t.Errorf("second Up returned %d, want 0 (all already applied)", count)
	}
}

func TestRunner_Version_ReturnsHighestApplied(t *testing.T) {
	dsn := requireDSN(t)
	dir := t.TempDir()

	writeSQLMigration(t, dir, 1, "v1", "CREATE TABLE IF NOT EXISTS _test_ver1 (id INT);", "DROP TABLE IF EXISTS _test_ver1;")
	writeSQLMigration(t, dir, 2, "v2", "CREATE TABLE IF NOT EXISTS _test_ver2 (id INT);", "DROP TABLE IF EXISTS _test_ver2;")

	r, err := NewRunner(dsn, dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	defer r.Close() //nolint:errcheck
	cleanupMigrationTable(t, r)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = r.db.ExecContext(ctx, `DROP TABLE IF EXISTS _test_ver1`)
		_, _ = r.db.ExecContext(ctx, `DROP TABLE IF EXISTS _test_ver2`)
	})

	ctx := context.Background()
	if err := r.EnsureTable(ctx); err != nil {
		t.Fatalf("EnsureTable: %v", err)
	}

	// Before any migrations: version must be 0.
	v, err := r.Version(ctx)
	if err != nil {
		t.Fatalf("Version (empty): %v", err)
	}
	if v != 0 {
		t.Errorf("Version (empty) = %d, want 0", v)
	}

	if _, err := r.Up(ctx); err != nil {
		t.Fatalf("Up: %v", err)
	}

	v, err = r.Version(ctx)
	if err != nil {
		t.Fatalf("Version (after Up): %v", err)
	}
	if v != 2 {
		t.Errorf("Version (after Up) = %d, want 2", v)
	}
}

// ---- Down -------------------------------------------------------------------

func TestRunner_Down_RollsBackLatest(t *testing.T) {
	dsn := requireDSN(t)
	dir := t.TempDir()

	writeSQLMigration(t, dir, 1, "first",
		"CREATE TABLE IF NOT EXISTS _test_down1 (id INT);",
		"DROP TABLE IF EXISTS _test_down1;",
	)
	writeSQLMigration(t, dir, 2, "second",
		"CREATE TABLE IF NOT EXISTS _test_down2 (id INT);",
		"DROP TABLE IF EXISTS _test_down2;",
	)

	r, err := NewRunner(dsn, dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	defer r.Close() //nolint:errcheck
	cleanupMigrationTable(t, r)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = r.db.ExecContext(ctx, `DROP TABLE IF EXISTS _test_down1`)
		_, _ = r.db.ExecContext(ctx, `DROP TABLE IF EXISTS _test_down2`)
	})

	ctx := context.Background()
	if _, err := r.Up(ctx); err != nil {
		t.Fatalf("Up: %v", err)
	}

	// Version should be 2 after Up.
	v, _ := r.Version(ctx)
	if v != 2 {
		t.Fatalf("Version before Down = %d, want 2", v)
	}

	if err := r.Down(ctx); err != nil {
		t.Fatalf("Down: %v", err)
	}

	// Version should now be 1.
	v, err = r.Version(ctx)
	if err != nil {
		t.Fatalf("Version after Down: %v", err)
	}
	if v != 1 {
		t.Errorf("Version after Down = %d, want 1", v)
	}
}

func TestRunner_Down_NoMigrations_IsNoop(t *testing.T) {
	dsn := requireDSN(t)
	dir := t.TempDir()

	// Write a migration file but don't apply it.
	writeSQLMigration(t, dir, 1, "noop",
		"CREATE TABLE IF NOT EXISTS _test_noop (id INT);",
		"DROP TABLE IF EXISTS _test_noop;",
	)

	r, err := NewRunner(dsn, dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	defer r.Close() //nolint:errcheck
	cleanupMigrationTable(t, r)

	ctx := context.Background()
	if err := r.EnsureTable(ctx); err != nil {
		t.Fatalf("EnsureTable: %v", err)
	}

	// Down with nothing applied should be a no-op (not an error).
	if err := r.Down(ctx); err != nil {
		t.Errorf("Down on empty state: unexpected error: %v", err)
	}
}

// ---- Status -----------------------------------------------------------------

func TestRunner_Status_ReturnsAppliedAndPending(t *testing.T) {
	dsn := requireDSN(t)
	dir := t.TempDir()

	writeSQLMigration(t, dir, 1, "applied_one",
		"CREATE TABLE IF NOT EXISTS _test_stat1 (id INT);",
		"DROP TABLE IF EXISTS _test_stat1;",
	)
	writeSQLMigration(t, dir, 2, "pending_two",
		"CREATE TABLE IF NOT EXISTS _test_stat2 (id INT);",
		"DROP TABLE IF EXISTS _test_stat2;",
	)

	r, err := NewRunner(dsn, dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	defer r.Close() //nolint:errcheck
	cleanupMigrationTable(t, r)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = r.db.ExecContext(ctx, `DROP TABLE IF EXISTS _test_stat1`)
		_, _ = r.db.ExecContext(ctx, `DROP TABLE IF EXISTS _test_stat2`)
	})

	ctx := context.Background()
	if err := r.EnsureTable(ctx); err != nil {
		t.Fatalf("EnsureTable: %v", err)
	}

	// Apply only the first migration.
	if err := r.applyUp(ctx, &Migration{Version: 1, Name: "applied_one",
		UpSQL: "CREATE TABLE IF NOT EXISTS _test_stat1 (id INT);"}); err != nil {
		t.Fatalf("applyUp: %v", err)
	}

	statuses, err := r.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("Status: got %d entries, want 2", len(statuses))
	}

	statusMap := make(map[int64]MigrationStatus, len(statuses))
	for _, s := range statuses {
		statusMap[s.Version] = s
	}

	if !statusMap[1].Applied {
		t.Error("version 1: Applied = false, want true")
	}
	if statusMap[2].Applied {
		t.Error("version 2: Applied = true, want false")
	}
}

// ---- StoreFingerprint -------------------------------------------------------

func TestRunner_StoreFingerprint_SetsOnMaxVersion(t *testing.T) {
	dsn := requireDSN(t)
	dir := t.TempDir()

	writeSQLMigration(t, dir, 1, "fp_v1",
		"CREATE TABLE IF NOT EXISTS _test_fp1 (id INT);",
		"DROP TABLE IF EXISTS _test_fp1;",
	)
	writeSQLMigration(t, dir, 2, "fp_v2",
		"CREATE TABLE IF NOT EXISTS _test_fp2 (id INT);",
		"DROP TABLE IF EXISTS _test_fp2;",
	)

	r, err := NewRunner(dsn, dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	defer r.Close() //nolint:errcheck
	cleanupMigrationTable(t, r)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = r.db.ExecContext(ctx, `DROP TABLE IF EXISTS _test_fp1`)
		_, _ = r.db.ExecContext(ctx, `DROP TABLE IF EXISTS _test_fp2`)
	})

	ctx := context.Background()
	if _, err := r.Up(ctx); err != nil {
		t.Fatalf("Up: %v", err)
	}

	const fp = "abc123"
	if err := r.StoreFingerprint(ctx, fp); err != nil {
		t.Fatalf("StoreFingerprint: %v", err)
	}

	// Verify: max-version row (2) has fingerprint set; version 1 has NULL.
	var fp1, fp2 *string
	if err := r.db.QueryRowContext(ctx,
		`SELECT fingerprint FROM _migrations WHERE version = 1`).Scan(&fp1); err != nil {
		t.Fatalf("query v1 fingerprint: %v", err)
	}
	if err := r.db.QueryRowContext(ctx,
		`SELECT fingerprint FROM _migrations WHERE version = 2`).Scan(&fp2); err != nil {
		t.Fatalf("query v2 fingerprint: %v", err)
	}

	if fp1 != nil {
		t.Errorf("version 1 fingerprint = %v, want nil (cleared)", fp1)
	}
	if fp2 == nil || *fp2 != fp {
		t.Errorf("version 2 fingerprint = %v, want %q", fp2, fp)
	}
}

func TestRunner_StoreFingerprint_ClearsPrevious(t *testing.T) {
	dsn := requireDSN(t)
	dir := t.TempDir()

	writeSQLMigration(t, dir, 1, "clear_v1",
		"CREATE TABLE IF NOT EXISTS _test_clear1 (id INT);",
		"DROP TABLE IF EXISTS _test_clear1;",
	)
	writeSQLMigration(t, dir, 2, "clear_v2",
		"CREATE TABLE IF NOT EXISTS _test_clear2 (id INT);",
		"DROP TABLE IF EXISTS _test_clear2;",
	)

	r, err := NewRunner(dsn, dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	defer r.Close() //nolint:errcheck
	cleanupMigrationTable(t, r)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = r.db.ExecContext(ctx, `DROP TABLE IF EXISTS _test_clear1`)
		_, _ = r.db.ExecContext(ctx, `DROP TABLE IF EXISTS _test_clear2`)
	})

	ctx := context.Background()
	if _, err := r.Up(ctx); err != nil {
		t.Fatalf("Up: %v", err)
	}

	// Store first fingerprint — lands on version 2.
	if err := r.StoreFingerprint(ctx, "first"); err != nil {
		t.Fatalf("first StoreFingerprint: %v", err)
	}

	// Store second fingerprint — must clear the first and set on version 2 again.
	if err := r.StoreFingerprint(ctx, "second"); err != nil {
		t.Fatalf("second StoreFingerprint: %v", err)
	}

	// Count rows with a non-NULL fingerprint: must be exactly 1.
	var cnt int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM _migrations WHERE fingerprint IS NOT NULL`).Scan(&cnt); err != nil {
		t.Fatalf("count fingerprints: %v", err)
	}
	if cnt != 1 {
		t.Errorf("fingerprint count = %d, want 1", cnt)
	}
}

// ---- EnsureTable / Applied --------------------------------------------------

func TestRunner_EnsureTable_Idempotent(t *testing.T) {
	dsn := requireDSN(t)
	dir := t.TempDir()

	r, err := NewRunner(dsn, dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	defer r.Close() //nolint:errcheck
	cleanupMigrationTable(t, r)

	ctx := context.Background()
	// Call EnsureTable twice — the second call must not fail.
	if err := r.EnsureTable(ctx); err != nil {
		t.Fatalf("first EnsureTable: %v", err)
	}
	if err := r.EnsureTable(ctx); err != nil {
		t.Fatalf("second EnsureTable (idempotent): %v", err)
	}
}

func TestRunner_Applied_EmptyAfterEnsure(t *testing.T) {
	dsn := requireDSN(t)
	dir := t.TempDir()

	r, err := NewRunner(dsn, dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	defer r.Close() //nolint:errcheck
	cleanupMigrationTable(t, r)

	ctx := context.Background()
	if err := r.EnsureTable(ctx); err != nil {
		t.Fatalf("EnsureTable: %v", err)
	}

	applied, err := r.Applied(ctx)
	if err != nil {
		t.Fatalf("Applied: %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("Applied (fresh table) = %v, want empty", applied)
	}
}
