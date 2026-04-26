package seed

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// All tests in this file that require a live database are guarded with
// TEST_DATABASE_URL. Set this env var to run the integration tests:
// TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/test_db

func requireDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	return dsn
}

// openDB opens a raw sql.DB for direct assertions inside integration tests.
func openDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// truncateSeedHistory drops then recreates the _seed_history table so each
// integration test starts from a clean state.
func truncateSeedHistory(t *testing.T, db *sql.DB) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS _seed_history`)
	})
}

// ---- Unit-level: Entry type -------------------------------------------------

func TestEntry_Fields(t *testing.T) {
	e := Entry{
		Name:     "fixtures/users.sql",
		Priority: 10,
		SQL:      "INSERT INTO users (id) VALUES (1);",
	}
	if e.Name != "fixtures/users.sql" {
		t.Errorf("Name = %q, want %q", e.Name, "fixtures/users.sql")
	}
	if e.Priority != 10 {
		t.Errorf("Priority = %d, want 10", e.Priority)
	}
	if e.SQL != "INSERT INTO users (id) VALUES (1);" {
		t.Errorf("SQL = %q, want %q", e.SQL, "INSERT INTO users (id) VALUES (1);")
	}
}

// ---- Unit-level: Status type ------------------------------------------------

func TestStatus_Fields(t *testing.T) {
	now := time.Now()
	s := Status{
		Name:      "fixtures/base.sql",
		Applied:   true,
		AppliedAt: now,
	}
	if s.Name != "fixtures/base.sql" {
		t.Errorf("Name = %q, want %q", s.Name, "fixtures/base.sql")
	}
	if !s.Applied {
		t.Error("Applied = false, want true")
	}
	if s.AppliedAt != now {
		t.Errorf("AppliedAt = %v, want %v", s.AppliedAt, now)
	}
}

// ---- Integration: Apply -----------------------------------------------------

func TestApply_EmptyEntries_ReturnsZero(t *testing.T) {
	dsn := requireDSN(t)

	ctx := context.Background()
	count, err := Apply(ctx, dsn, []Entry{})
	if err != nil {
		t.Fatalf("Apply (empty): unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("Apply (empty) = %d, want 0", count)
	}
}

func TestApply_SingleEntry_Applied(t *testing.T) {
	dsn := requireDSN(t)
	db := openDB(t, dsn)
	truncateSeedHistory(t, db)

	ctx := context.Background()
	entries := []Entry{
		{Name: "test/seed_one", Priority: 0, SQL: "SELECT 1;"},
	}
	count, err := Apply(ctx, dsn, entries)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if count != 1 {
		t.Errorf("Apply = %d, want 1", count)
	}
}

func TestApply_AlreadyApplied_Skipped(t *testing.T) {
	dsn := requireDSN(t)
	db := openDB(t, dsn)
	truncateSeedHistory(t, db)

	ctx := context.Background()
	entries := []Entry{
		{Name: "test/seed_skip", Priority: 0, SQL: "SELECT 1;"},
	}

	// First apply.
	if _, err := Apply(ctx, dsn, entries); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	// Second apply — must skip the already-applied entry.
	count, err := Apply(ctx, dsn, entries)
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	if count != 0 {
		t.Errorf("second Apply = %d, want 0 (already applied)", count)
	}
}

func TestApply_PriorityOrder(t *testing.T) {
	dsn := requireDSN(t)
	db := openDB(t, dsn)
	truncateSeedHistory(t, db)

	// Create a temp table to record insertion order.
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `
		CREATE TEMP TABLE IF NOT EXISTS _seed_order (
			name TEXT,
			seq  SERIAL
		)
	`)
	if err != nil {
		t.Fatalf("create temp table: %v", err)
	}

	// Entries given in reverse priority order — Apply must execute lowest priority first.
	entries := []Entry{
		{Name: "test/high_priority", Priority: 10, SQL: "INSERT INTO _seed_order (name) VALUES ('high');"},
		{Name: "test/low_priority", Priority: 1, SQL: "INSERT INTO _seed_order (name) VALUES ('low');"},
	}

	count, err := Apply(ctx, dsn, entries)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if count != 2 {
		t.Fatalf("Apply = %d, want 2", count)
	}

	// Verify order: 'low' (priority 1) should have a lower seq than 'high' (priority 10).
	rows, err := db.QueryContext(ctx, `SELECT name FROM _seed_order ORDER BY seq`)
	if err != nil {
		t.Fatalf("query order: %v", err)
	}
	defer rows.Close() //nolint:errcheck

	var order []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		order = append(order, n)
	}
	if len(order) != 2 {
		t.Fatalf("order rows: got %d, want 2", len(order))
	}
	if order[0] != "low" || order[1] != "high" {
		t.Errorf("order = %v, want [low high]", order)
	}
}

func TestApply_MultipleEntries_CountMatches(t *testing.T) {
	dsn := requireDSN(t)
	db := openDB(t, dsn)
	truncateSeedHistory(t, db)

	ctx := context.Background()
	entries := []Entry{
		{Name: "test/a", Priority: 0, SQL: "SELECT 1;"},
		{Name: "test/b", Priority: 1, SQL: "SELECT 2;"},
		{Name: "test/c", Priority: 2, SQL: "SELECT 3;"},
	}
	count, err := Apply(ctx, dsn, entries)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if count != 3 {
		t.Errorf("Apply = %d, want 3", count)
	}
}

// ---- Integration: GetStatus -------------------------------------------------

func TestGetStatus_KnownSeedsReturnsStatusForAll(t *testing.T) {
	dsn := requireDSN(t)
	db := openDB(t, dsn)
	truncateSeedHistory(t, db)

	ctx := context.Background()
	entries := []Entry{
		{Name: "test/status_one", Priority: 0, SQL: "SELECT 1;"},
	}

	// Apply one of two known seeds.
	if _, err := Apply(ctx, dsn, entries); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	known := []string{"test/status_one", "test/status_two"}
	statuses, err := GetStatus(ctx, dsn, known)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("GetStatus: got %d, want 2", len(statuses))
	}

	statusMap := make(map[string]Status, len(statuses))
	for _, s := range statuses {
		statusMap[s.Name] = s
	}

	if !statusMap["test/status_one"].Applied {
		t.Error("test/status_one: Applied = false, want true")
	}
	if statusMap["test/status_two"].Applied {
		t.Error("test/status_two: Applied = true, want false")
	}
}

func TestGetStatus_EmptyKnown_ReturnsEmpty(t *testing.T) {
	dsn := requireDSN(t)
	db := openDB(t, dsn)
	truncateSeedHistory(t, db)

	ctx := context.Background()
	// Ensure history table exists.
	if _, err := Apply(ctx, dsn, []Entry{}); err != nil {
		t.Fatalf("Apply (empty): %v", err)
	}

	statuses, err := GetStatus(ctx, dsn, []string{})
	if err != nil {
		t.Fatalf("GetStatus (empty known): %v", err)
	}
	if len(statuses) != 0 {
		t.Errorf("GetStatus (empty known) = %v, want []", statuses)
	}
}

func TestGetStatus_AppliedAt_NonZero(t *testing.T) {
	dsn := requireDSN(t)
	db := openDB(t, dsn)
	truncateSeedHistory(t, db)

	ctx := context.Background()
	entries := []Entry{
		{Name: "test/applied_at", Priority: 0, SQL: "SELECT 1;"},
	}
	if _, err := Apply(ctx, dsn, entries); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	statuses, err := GetStatus(ctx, dsn, []string{"test/applied_at"})
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("GetStatus: got %d, want 1", len(statuses))
	}
	if statuses[0].AppliedAt.IsZero() {
		t.Error("AppliedAt is zero for an applied seed — should be a real timestamp")
	}
}

// ---- Integration: Reset -----------------------------------------------------

func TestReset_ClearsSeedHistory(t *testing.T) {
	dsn := requireDSN(t)
	db := openDB(t, dsn)
	truncateSeedHistory(t, db)

	ctx := context.Background()
	entries := []Entry{
		{Name: "test/reset_me", Priority: 0, SQL: "SELECT 1;"},
	}

	// Apply a seed.
	if _, err := Apply(ctx, dsn, entries); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// Reset.
	if err := Reset(ctx, dsn); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	// After reset, applying the same entry should succeed (not be skipped).
	count, err := Apply(ctx, dsn, entries)
	if err != nil {
		t.Fatalf("Apply after Reset: %v", err)
	}
	if count != 1 {
		t.Errorf("Apply after Reset = %d, want 1", count)
	}
}

func TestReset_EmptyHistory_IsNoop(t *testing.T) {
	dsn := requireDSN(t)
	db := openDB(t, dsn)
	truncateSeedHistory(t, db)

	ctx := context.Background()
	// Ensure the table exists but is empty.
	if _, err := Apply(ctx, dsn, []Entry{}); err != nil {
		t.Fatalf("Apply (empty): %v", err)
	}

	// Reset on empty table must not error.
	if err := Reset(ctx, dsn); err != nil {
		t.Fatalf("Reset (empty): %v", err)
	}
}

// ---- Integration: DSN validation --------------------------------------------

func TestApply_InvalidDSN_ReturnsError(t *testing.T) {
	// This test does NOT require TEST_DATABASE_URL — it uses a deliberately bad DSN.
	ctx := context.Background()
	_, err := Apply(ctx, "postgres://invalid:invalid@localhost:0/nonexistent", []Entry{
		{Name: "test/bad_dsn", Priority: 0, SQL: "SELECT 1;"},
	})
	if err == nil {
		t.Fatal("Apply with invalid DSN: expected error, got nil")
	}
}

func TestGetStatus_InvalidDSN_ReturnsError(t *testing.T) {
	ctx := context.Background()
	_, err := GetStatus(ctx, "postgres://invalid:invalid@localhost:0/nonexistent", []string{"test/any"})
	if err == nil {
		t.Fatal("GetStatus with invalid DSN: expected error, got nil")
	}
}

func TestReset_InvalidDSN_ReturnsError(t *testing.T) {
	ctx := context.Background()
	err := Reset(ctx, "postgres://invalid:invalid@localhost:0/nonexistent")
	if err == nil {
		t.Fatal("Reset with invalid DSN: expected error, got nil")
	}
}

// ---- Helper: verify historyDDL constant -------------------------------------

func TestHistoryDDL_ContainsExpectedTable(t *testing.T) {
	// historyDDL is a package-level constant — verify it names the right table.
	expected := "_seed_history"
	if len(historyDDL) == 0 {
		t.Fatal("historyDDL is empty")
	}
	found := false
	for i := 0; i <= len(historyDDL)-len(expected); i++ {
		if historyDDL[i:i+len(expected)] == expected {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("historyDDL does not contain table name %q: %s", expected, historyDDL)
	}
}

// ---- Integration: Apply with $$ SQL (dollar-quote safety) -------------------

func TestApply_DollarQuotedSQL_ParsedCorrectly(t *testing.T) {
	dsn := requireDSN(t)
	db := openDB(t, dsn)
	truncateSeedHistory(t, db)

	// SQL containing $$ that would be incorrectly split by a naive semicolon splitter.
	sql := fmt.Sprintf(`CREATE OR REPLACE FUNCTION _test_seed_fn_%d()
RETURNS void AS $$
BEGIN
    NULL;
END;
$$ LANGUAGE plpgsql;`, time.Now().UnixNano())

	ctx := context.Background()
	entries := []Entry{
		{Name: "test/dollar_quote", Priority: 0, SQL: sql},
	}

	count, err := Apply(ctx, dsn, entries)
	if err != nil {
		t.Fatalf("Apply (dollar-quoted SQL): %v", err)
	}
	if count != 1 {
		t.Errorf("Apply (dollar-quoted SQL) = %d, want 1", count)
	}
}
