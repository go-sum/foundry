package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- SplitStatements --------------------------------------------------------

func TestSplitStatements_SingleStatement(t *testing.T) {
	stmts := SplitStatements("SELECT 1;")
	if len(stmts) != 1 {
		t.Fatalf("got %d statements, want 1: %v", len(stmts), stmts)
	}
	if stmts[0] != "SELECT 1;" {
		t.Errorf("stmts[0] = %q, want %q", stmts[0], "SELECT 1;")
	}
}

func TestSplitStatements_TwoStatements(t *testing.T) {
	sql := "SELECT 1;\nSELECT 2;"
	stmts := SplitStatements(sql)
	if len(stmts) != 2 {
		t.Fatalf("got %d statements, want 2: %v", len(stmts), stmts)
	}
	if stmts[0] != "SELECT 1;" {
		t.Errorf("stmts[0] = %q, want %q", stmts[0], "SELECT 1;")
	}
	if stmts[1] != "SELECT 2;" {
		t.Errorf("stmts[1] = %q, want %q", stmts[1], "SELECT 2;")
	}
}

func TestSplitStatements_DollarQuotedBodyNotSplit(t *testing.T) {
	// Semicolons inside $$ should NOT split the statement.
	sql := `CREATE OR REPLACE FUNCTION fn()
RETURNS TRIGGER AS $$
BEGIN
    NEW.x = 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;`

	stmts := SplitStatements(sql)
	if len(stmts) != 1 {
		t.Fatalf("got %d statements, want 1 (dollar-quote blocks must not split): %v", len(stmts), stmts)
	}
	if !strings.Contains(stmts[0], "RETURN NEW") {
		t.Errorf("statement missing body content: %q", stmts[0])
	}
}

func TestSplitStatements_Empty(t *testing.T) {
	stmts := SplitStatements("")
	if len(stmts) != 0 {
		t.Fatalf("got %d statements, want 0: %v", len(stmts), stmts)
	}
}

func TestSplitStatements_WhitespaceOnly(t *testing.T) {
	stmts := SplitStatements("   \n\t  ")
	if len(stmts) != 0 {
		t.Fatalf("got %d statements, want 0: %v", len(stmts), stmts)
	}
}

func TestSplitStatements_NoTrailingSemicolon(t *testing.T) {
	// A statement with no trailing semicolon should still be included.
	stmts := SplitStatements("SELECT 1")
	if len(stmts) != 1 {
		t.Fatalf("got %d statements, want 1: %v", len(stmts), stmts)
	}
	if stmts[0] != "SELECT 1" {
		t.Errorf("stmts[0] = %q, want %q", stmts[0], "SELECT 1")
	}
}

func TestSplitStatements_MultipleWithDollarQuote(t *testing.T) {
	// Two statements: one normal and one with a $$ block.
	sql := `CREATE TABLE foo (id INT);

CREATE OR REPLACE FUNCTION fn()
RETURNS VOID AS $$
BEGIN
    NULL;
END;
$$ LANGUAGE plpgsql;`

	stmts := SplitStatements(sql)
	if len(stmts) != 2 {
		t.Fatalf("got %d statements, want 2: %v", len(stmts), stmts)
	}
	if !strings.Contains(stmts[0], "CREATE TABLE foo") {
		t.Errorf("stmts[0] missing CREATE TABLE: %q", stmts[0])
	}
	if !strings.Contains(stmts[1], "CREATE OR REPLACE FUNCTION") {
		t.Errorf("stmts[1] missing CREATE FUNCTION: %q", stmts[1])
	}
}

func TestSplitStatements_BlankLinesAndComments(t *testing.T) {
	// Blank statements (whitespace-only between semicolons) must be omitted.
	sql := "SELECT 1;\n\n   \nSELECT 2;"
	stmts := SplitStatements(sql)
	if len(stmts) != 2 {
		t.Fatalf("got %d statements, want 2 (blank statements must be omitted): %v", len(stmts), stmts)
	}
}

// ---- ParseFile --------------------------------------------------------------

func writeTempMigration(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeTempMigration %q: %v", name, err)
	}
	return path
}

func TestParseFile_ValidMigration(t *testing.T) {
	dir := t.TempDir()
	content := `-- +migrate Up
CREATE TABLE foo (id INT);

-- +migrate Down
DROP TABLE IF EXISTS foo;
`
	path := writeTempMigration(t, dir, "00003_add_foo.sql", content)

	m, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: unexpected error: %v", err)
	}
	if m.Version != 3 {
		t.Errorf("Version = %d, want 3", m.Version)
	}
	if m.Name != "add_foo" {
		t.Errorf("Name = %q, want %q", m.Name, "add_foo")
	}
	if !strings.Contains(m.UpSQL, "CREATE TABLE foo") {
		t.Errorf("UpSQL missing CREATE TABLE: %q", m.UpSQL)
	}
	if !strings.Contains(m.DownSQL, "DROP TABLE IF EXISTS foo") {
		t.Errorf("DownSQL missing DROP TABLE: %q", m.DownSQL)
	}
	if m.Source != path {
		t.Errorf("Source = %q, want %q", m.Source, path)
	}
}

func TestParseFile_VersionExtraction(t *testing.T) {
	tests := []struct {
		filename string
		version  int64
		name     string
	}{
		{"00001_initial.sql", 1, "initial"},
		{"00042_add_users.sql", 42, "add_users"},
		{"00100_big_version.sql", 100, "big_version"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			dir := t.TempDir()
			content := "-- +migrate Up\nSELECT 1;\n-- +migrate Down\nSELECT 1;\n"
			path := writeTempMigration(t, dir, tt.filename, content)

			m, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			if m.Version != tt.version {
				t.Errorf("Version = %d, want %d", m.Version, tt.version)
			}
			if m.Name != tt.name {
				t.Errorf("Name = %q, want %q", m.Name, tt.name)
			}
		})
	}
}

func TestParseFile_NonexistentFile(t *testing.T) {
	_, err := ParseFile("/does/not/exist/00001_missing.sql")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestParseFile_InvalidVersionFormat(t *testing.T) {
	dir := t.TempDir()
	// Filename without numeric prefix — version parsing should fail.
	path := writeTempMigration(t, dir, "no_version.sql", "-- +migrate Up\nSELECT 1;\n")

	_, err := ParseFile(path)
	if err == nil {
		t.Fatal("expected error for invalid version format, got nil")
	}
}

func TestParseFile_UpSQLPreservedExactly(t *testing.T) {
	dir := t.TempDir()
	upSQL := "CREATE TABLE widgets (\n    id UUID PRIMARY KEY\n);"
	content := "-- +migrate Up\n" + upSQL + "\n-- +migrate Down\nDROP TABLE IF EXISTS widgets;\n"
	path := writeTempMigration(t, dir, "00001_widgets.sql", content)

	m, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if m.UpSQL != upSQL {
		t.Errorf("UpSQL:\ngot  %q\nwant %q", m.UpSQL, upSQL)
	}
}

func TestParseFile_EmptySections(t *testing.T) {
	dir := t.TempDir()
	// A migration with empty Up and Down sections.
	content := "-- +migrate Up\n-- +migrate Down\n"
	path := writeTempMigration(t, dir, "00001_empty_sections.sql", content)

	m, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if m.UpSQL != "" {
		t.Errorf("UpSQL = %q, want empty string", m.UpSQL)
	}
	if m.DownSQL != "" {
		t.Errorf("DownSQL = %q, want empty string", m.DownSQL)
	}
}

func TestParseFile_ContentBeforeFirstMarker_Ignored(t *testing.T) {
	dir := t.TempDir()
	// Lines before the first marker should be ignored (treated as comments/preamble).
	content := "-- This is a migration\n-- +migrate Up\nSELECT 1;\n-- +migrate Down\nSELECT 2;\n"
	path := writeTempMigration(t, dir, "00001_preamble.sql", content)

	m, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if m.UpSQL != "SELECT 1;" {
		t.Errorf("UpSQL = %q, want %q", m.UpSQL, "SELECT 1;")
	}
}

// ---- ParseDir ---------------------------------------------------------------

func TestParseDir_SortedByVersion(t *testing.T) {
	dir := t.TempDir()
	// Write files in reverse order to verify sorting.
	for _, name := range []string{"00003_c.sql", "00001_a.sql", "00002_b.sql"} {
		writeTempMigration(t, dir, name, "-- +migrate Up\nSELECT 1;\n-- +migrate Down\nSELECT 1;\n")
	}

	migrations, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if len(migrations) != 3 {
		t.Fatalf("got %d migrations, want 3", len(migrations))
	}
	versions := []int64{migrations[0].Version, migrations[1].Version, migrations[2].Version}
	expected := []int64{1, 2, 3}
	for i, v := range versions {
		if v != expected[i] {
			t.Errorf("migrations[%d].Version = %d, want %d", i, v, expected[i])
		}
	}
}

func TestParseDir_SkipsDotPrefix(t *testing.T) {
	dir := t.TempDir()
	// Write a valid migration and a .schema/ subdirectory entry.
	writeTempMigration(t, dir, "00001_valid.sql", "-- +migrate Up\nSELECT 1;\n-- +migrate Down\nSELECT 1;\n")

	// Create .schema directory with a snapshot file (should be skipped).
	schemaDir := filepath.Join(dir, ".schema")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(schemaDir, "base.sql"), []byte("CREATE EXTENSION IF NOT EXISTS citext;"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	migrations, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if len(migrations) != 1 {
		t.Fatalf("got %d migrations, want 1 (dot-prefixed dirs must be skipped)", len(migrations))
	}
	if migrations[0].Version != 1 {
		t.Errorf("migrations[0].Version = %d, want 1", migrations[0].Version)
	}
}

func TestParseDir_SkipsNonSQLFiles(t *testing.T) {
	dir := t.TempDir()
	writeTempMigration(t, dir, "00001_valid.sql", "-- +migrate Up\nSELECT 1;\n-- +migrate Down\nSELECT 1;\n")

	// Write a .txt and .json file that should be skipped.
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("notes"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	migrations, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if len(migrations) != 1 {
		t.Fatalf("got %d migrations, want 1 (non-sql files must be skipped)", len(migrations))
	}
}

func TestParseDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	migrations, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir on empty dir: unexpected error: %v", err)
	}
	if len(migrations) != 0 {
		t.Fatalf("got %d migrations, want 0", len(migrations))
	}
}

func TestParseDir_NonexistentDir(t *testing.T) {
	_, err := ParseDir("/does/not/exist/migrations")
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
}

func TestParseDir_SkipsSubdirectories(t *testing.T) {
	dir := t.TempDir()
	writeTempMigration(t, dir, "00001_valid.sql", "-- +migrate Up\nSELECT 1;\n-- +migrate Down\nSELECT 1;\n")

	// Create a regular subdirectory (non-dot-prefixed) — should also be skipped.
	subDir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Write a SQL file inside the subdirectory — must not be included.
	writeTempMigration(t, subDir, "00002_nested.sql", "-- +migrate Up\nSELECT 2;\n-- +migrate Down\nSELECT 2;\n")

	migrations, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if len(migrations) != 1 {
		t.Fatalf("got %d migrations, want 1 (subdirectories must not be recursed into)", len(migrations))
	}
}
