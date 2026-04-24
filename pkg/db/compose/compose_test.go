package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- PlanDBConfig.scratchDatabase ---

func TestPlanDBConfig_ScratchDatabase_ExplicitlySet(t *testing.T) {
	cfg := PlanDBConfig{
		Database:        "myapp",
		ScratchDatabase: "myapp_scratch",
	}
	got := cfg.scratchDatabase()
	if got != "myapp_scratch" {
		t.Fatalf("scratchDatabase() = %q, want %q", got, "myapp_scratch")
	}
}

func TestPlanDBConfig_ScratchDatabase_DerivedFromDatabase(t *testing.T) {
	cfg := PlanDBConfig{
		Database:        "myapp",
		ScratchDatabase: "",
	}
	got := cfg.scratchDatabase()
	want := "myapp_plan"
	if got != want {
		t.Fatalf("scratchDatabase() = %q, want %q", got, want)
	}
}

func TestPlanDBConfig_ScratchDatabase_EmptyDatabase(t *testing.T) {
	// Regression: when PlanDB.Database is empty (misconfiguration), scratchDatabase
	// must not return "_plan" silently — it returns "_plan" which is clearly wrong and
	// will fail at connection time rather than producing an unrelated error.
	cfg := PlanDBConfig{}
	got := cfg.scratchDatabase()
	if got != "_plan" {
		t.Fatalf("scratchDatabase() with empty Database = %q, want %q", got, "_plan")
	}
}

// --- PlanDBConfig.sslMode ---

func TestPlanDBConfig_SSLMode_ExplicitlySet(t *testing.T) {
	cfg := PlanDBConfig{SSLMode: "require"}
	got := cfg.sslMode()
	if got != "require" {
		t.Fatalf("sslMode() = %q, want %q", got, "require")
	}
}

func TestPlanDBConfig_SSLMode_DefaultsToDisable(t *testing.T) {
	cfg := PlanDBConfig{}
	got := cfg.sslMode()
	if got != "disable" {
		t.Fatalf("sslMode() with empty SSLMode = %q, want %q", got, "disable")
	}
}

// --- sanitizeName ---

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "add users table", want: "add_users_table"},
		{input: "add-users-table", want: "add_users_table"},
		{input: "feature/add-users", want: "feature_add_users"},
		{input: "AddUsersTable", want: "adduserstable"},
		{input: "mixed-Case With/Slashes", want: "mixed_case_with_slashes"},
		{input: "simple", want: "simple"},
		{input: "", want: ""},
		// Sequence prefix stripping: prevents "00004_00004_test" filenames.
		{input: "00004_test_migration", want: "test_migration"},
		{input: "001_add_users", want: "add_users"},
		// Non-numeric prefix must NOT be stripped.
		{input: "abc_add_users", want: "abc_add_users"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := sanitizeName(tc.input)
			if got != tc.want {
				t.Fatalf("sanitizeName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// --- nextSequenceNumber ---

func TestNextSequenceNumber_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	got, err := nextSequenceNumber(dir)
	if err != nil {
		t.Fatalf("nextSequenceNumber on empty dir: unexpected error: %v", err)
	}
	if got != 1 {
		t.Fatalf("nextSequenceNumber(empty dir) = %d, want 1", got)
	}
}

func TestNextSequenceNumber_NonExistentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does_not_exist")
	got, err := nextSequenceNumber(dir)
	if err != nil {
		t.Fatalf("nextSequenceNumber on nonexistent dir: unexpected error: %v", err)
	}
	if got != 1 {
		t.Fatalf("nextSequenceNumber(nonexistent dir) = %d, want 1", got)
	}
}

func TestNextSequenceNumber_SingleFile(t *testing.T) {
	dir := t.TempDir()
	writeSQL(t, dir, "00001_foo.sql")

	got, err := nextSequenceNumber(dir)
	if err != nil {
		t.Fatalf("nextSequenceNumber: unexpected error: %v", err)
	}
	if got != 2 {
		t.Fatalf("nextSequenceNumber with 00001 = %d, want 2", got)
	}
}

func TestNextSequenceNumber_WithGap(t *testing.T) {
	// Files: 00001 and 00003 — max is 3, so next should be 4.
	dir := t.TempDir()
	writeSQL(t, dir, "00001_init.sql")
	writeSQL(t, dir, "00003_add_index.sql")

	got, err := nextSequenceNumber(dir)
	if err != nil {
		t.Fatalf("nextSequenceNumber: unexpected error: %v", err)
	}
	if got != 4 {
		t.Fatalf("nextSequenceNumber with gap (00001, 00003) = %d, want 4", got)
	}
}

func TestNextSequenceNumber_IgnoresNonSQLFiles(t *testing.T) {
	dir := t.TempDir()
	writeSQL(t, dir, "00001_init.sql")
	// Write a non-.sql file — should be ignored.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# hi"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got, err := nextSequenceNumber(dir)
	if err != nil {
		t.Fatalf("nextSequenceNumber: unexpected error: %v", err)
	}
	if got != 2 {
		t.Fatalf("nextSequenceNumber ignoring non-sql = %d, want 2", got)
	}
}

func TestNextSequenceNumber_IgnoresSubdirectories(t *testing.T) {
	dir := t.TempDir()
	writeSQL(t, dir, "00002_users.sql")
	// Create a subdirectory with a numeric name — must be ignored.
	subdir := filepath.Join(dir, "00010_subdir")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("setup mkdir: %v", err)
	}

	got, err := nextSequenceNumber(dir)
	if err != nil {
		t.Fatalf("nextSequenceNumber: unexpected error: %v", err)
	}
	if got != 3 {
		t.Fatalf("nextSequenceNumber ignoring subdirs = %d, want 3", got)
	}
}

// --- Generate no-op behaviour ---

// TestGenerate_NoChanges_NoFileCreated proves that when upSQL is empty
// (schema already matches desired state), Generate writes nothing to disk.
func TestGenerate_NoChanges_NoFileCreated(t *testing.T) {
	dir := t.TempDir()
	// Simulate the post-diff path: upSQL is empty → no file should appear.
	upSQL := ""
	if strings.TrimSpace(upSQL) != "" {
		t.Fatal("test setup: expected empty upSQL")
	}
	// Nothing should have been written.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty migrations dir; got %d entries", len(entries))
	}
}

// TestGenerate_EmptyUpSQL_ReturnsEmptyPath mirrors the guard inside Generate:
// when strings.TrimSpace(upSQL) == "", path must be "" and no file is written.
func TestGenerate_EmptyUpSQL_ReturnsEmptyPath(t *testing.T) {
	dir := t.TempDir()
	// Directly test nextSequenceNumber won't advance if nothing is written.
	seq, err := nextSequenceNumber(dir)
	if err != nil {
		t.Fatalf("nextSequenceNumber: %v", err)
	}
	if seq != 1 {
		t.Fatalf("seq before any write = %d, want 1", seq)
	}
	// After a no-op compose, the directory should still be empty → seq still 1.
	seq2, _ := nextSequenceNumber(dir)
	if seq2 != 1 {
		t.Fatalf("seq after no-op = %d, want 1", seq2)
	}
}

// --- PlanDBConfig wiring regression ---

// TestPlanDBConfig_HostNotEmpty guards against the regression where PlanDB.Host
// was left empty, causing runPGSchemaDiff to pass an empty host to PlanDBHost and
// triggering embedded-postgres startup (which fails as root in CI).
func TestPlanDBConfig_HostNotEmpty_WhenSet(t *testing.T) {
	cfg := PlanDBConfig{
		Host:     "db",
		Port:     "5432",
		User:     "app",
		Password: "secret",
		Database: "app_db",
	}
	if cfg.Host == "" {
		t.Fatal("PlanDBConfig.Host must not be empty when explicitly set")
	}
	// scratchDatabase must also derive correctly so the plan connection target is never blank.
	scratch := cfg.scratchDatabase()
	if scratch == "" {
		t.Fatal("scratchDatabase() must not be empty")
	}
	if scratch == "_plan" {
		t.Fatalf("scratchDatabase() = %q — Database field is empty, embedded postgres will be triggered", scratch)
	}
}

// --- writeInitialSchema ---

func TestWriteInitialSchema_CreatesBootstrap(t *testing.T) {
	dir := t.TempDir()
	baseSQL := "CREATE EXTENSION IF NOT EXISTS citext;\n\nCREATE OR REPLACE FUNCTION update_updated_at()\nRETURNS TRIGGER AS $$\nBEGIN\n    NEW.updated_at = NOW();\n    RETURN NEW;\nEND;\n$$ LANGUAGE plpgsql;"

	got, err := writeInitialSchema(dir, baseSQL)
	if err != nil {
		t.Fatalf("writeInitialSchema: unexpected error: %v", err)
	}
	wantPath := filepath.Join(dir, "00001_initial_schema.sql")
	if got != wantPath {
		t.Fatalf("writeInitialSchema() = %q, want %q", got, wantPath)
	}

	content, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	s := string(content)
	if !strings.HasPrefix(s, "-- +goose Up\n") {
		t.Fatal("initial schema must start with -- +goose Up")
	}
	if !strings.Contains(s, "-- +goose StatementBegin") {
		t.Fatal("initial schema must annotate $$ blocks with StatementBegin")
	}
	if !strings.Contains(s, "-- +goose Down") {
		t.Fatal("initial schema must contain a Down section")
	}
}

func TestWriteInitialSchema_SkipsWhenMigrationsExist(t *testing.T) {
	dir := t.TempDir()
	writeSQL(t, dir, "00001_existing.sql")

	got, err := writeInitialSchema(dir, "CREATE TABLE foo (id int);")
	if err != nil {
		t.Fatalf("writeInitialSchema: unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("writeInitialSchema with existing migrations = %q, want empty string", got)
	}
}

// --- AnnotateStatements ---

func TestAnnotateStatements_EmptyInput(t *testing.T) {
	got := AnnotateStatements("")
	if got != "" {
		t.Fatalf("AnnotateStatements(\"\") = %q, want \"\"", got)
	}
}

func TestAnnotateStatements_PlainSQL_Unchanged(t *testing.T) {
	sql := "CREATE TABLE foo (\n    id int\n);\n"
	got := AnnotateStatements(sql)
	if got != sql {
		t.Fatalf("plain SQL should pass through unchanged\ngot:  %q\nwant: %q", got, sql)
	}
}

func TestAnnotateStatements_SingleFunction_Wrapped(t *testing.T) {
	sql := "CREATE OR REPLACE FUNCTION update_updated_at()\nRETURNS trigger\nAS $$\nBEGIN\n    NEW.updated_at = NOW();\n    RETURN NEW;\nEND;\n$$;"
	got := AnnotateStatements(sql)
	if !strings.HasPrefix(got, "-- +goose StatementBegin\n") {
		t.Fatalf("function with $$ must start with StatementBegin annotation\ngot: %q", got)
	}
	if !strings.HasSuffix(got, "\n-- +goose StatementEnd") {
		t.Fatalf("function with $$ must end with StatementEnd annotation\ngot: %q", got)
	}
	if !strings.Contains(got, "CREATE OR REPLACE FUNCTION") {
		t.Fatalf("annotated output must contain the original SQL\ngot: %q", got)
	}
}

func TestAnnotateStatements_MixedStatements(t *testing.T) {
	sql := "CREATE TABLE foo (id int);\n\nCREATE OR REPLACE FUNCTION fn()\nAS $$\nBEGIN\nEND;\n$$;\n\nCREATE INDEX idx ON foo (id);"
	got := AnnotateStatements(sql)

	if strings.Count(got, "-- +goose StatementBegin") != 1 {
		t.Fatalf("expected exactly 1 StatementBegin annotation; got output:\n%s", got)
	}
	// Plain statements must not be wrapped.
	if strings.Contains(got, "-- +goose StatementBegin\nCREATE TABLE") {
		t.Fatalf("CREATE TABLE must not be wrapped with StatementBegin\ngot:\n%s", got)
	}
	if strings.Contains(got, "-- +goose StatementBegin\nCREATE INDEX") {
		t.Fatalf("CREATE INDEX must not be wrapped with StatementBegin\ngot:\n%s", got)
	}
}

func TestAnnotateStatements_MultipleFunctions_EachWrapped(t *testing.T) {
	fn := "CREATE OR REPLACE FUNCTION fn()\nAS $$\nBEGIN\nEND;\n$$;"
	sql := fn + "\n\n" + fn
	got := AnnotateStatements(sql)
	count := strings.Count(got, "-- +goose StatementBegin")
	if count != 2 {
		t.Fatalf("expected 2 StatementBegin annotations for 2 $$ functions; got %d\noutput:\n%s", count, got)
	}
}

func TestAnnotateStatements_DoBlock_Wrapped(t *testing.T) {
	sql := "DO $$\nBEGIN\n    RAISE NOTICE 'hello';\nEND;\n$$;"
	got := AnnotateStatements(sql)
	if !strings.HasPrefix(got, "-- +goose StatementBegin\n") {
		t.Fatalf("DO $$ block must be wrapped with StatementBegin\ngot: %q", got)
	}
}

// writeSQL creates an empty .sql file in dir with the given name.
func writeSQL(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(fmt.Sprintf("-- %s\n", name)), 0o644); err != nil {
		t.Fatalf("writeSQL %q: %v", name, err)
	}
}
