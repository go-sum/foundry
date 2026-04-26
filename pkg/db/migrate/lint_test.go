package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

// writeMigration writes a .sql migration file with the given SQL in the Up section.
func writeMigration(t *testing.T, dir, name, upSQL string) string {
	t.Helper()
	content := "-- +migrate Up\n" + upSQL + "\n-- +migrate Down\n-- nothing\n"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeMigration %q: %v", name, err)
	}
	return path
}

// writeDownOnlyMigration writes a file where the dangerous SQL is ONLY in the Down section.
func writeDownOnlyMigration(t *testing.T, dir, name, downSQL string) string {
	t.Helper()
	content := "-- +migrate Up\n-- nothing here\n-- +migrate Down\n" + downSQL + "\n"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeDownOnlyMigration %q: %v", name, err)
	}
	return path
}

// countByRule counts results with the given rule name.
func countByRule(results []LintResult, rule string) int {
	n := 0
	for _, r := range results {
		if r.Rule == rule {
			n++
		}
	}
	return n
}

// hasRule reports whether any result matches the given rule.
func hasRule(results []LintResult, rule string) bool {
	return countByRule(results, rule) > 0
}

func TestLint_EmptyUpSection_NoResults(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "00001_empty.sql", "")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Lint on empty Up section = %v, want no results", results)
	}
}

func TestLint_DropColumn_Warning(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "00001_drop_col.sql", "ALTER TABLE foo DROP COLUMN bar;")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if !hasRule(results, "drop-column") {
		t.Fatalf("expected drop-column warning, got %v", results)
	}
	r := results[0]
	if r.Severity != "warning" {
		t.Fatalf("drop-column severity = %q, want %q", r.Severity, "warning")
	}
}

func TestLint_AlterColumnType_Warning(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "00001_alter_type.sql", "ALTER TABLE foo ALTER COLUMN bar TYPE TEXT;")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if !hasRule(results, "alter-column-type") {
		t.Fatalf("expected alter-column-type warning, got %v", results)
	}
	r := results[0]
	if r.Severity != "warning" {
		t.Fatalf("alter-column-type severity = %q, want %q", r.Severity, "warning")
	}
}

func TestLint_NotNullNoDefault_InAlterTable_Warning(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "00001_notnull.sql", "ALTER TABLE foo ADD COLUMN bar TEXT NOT NULL;")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if !hasRule(results, "not-null-no-default") {
		t.Fatalf("expected not-null-no-default warning, got %v", results)
	}
}

func TestLint_NotNullNoDefault_InCreateTable_NoWarning(t *testing.T) {
	// The 3B fix: NOT NULL inside a CREATE TABLE block must not fire.
	dir := t.TempDir()
	writeMigration(t, dir, "00001_create.sql",
		"CREATE TABLE foo (\n    bar TEXT NOT NULL\n);")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if hasRule(results, "not-null-no-default") {
		t.Fatalf("not-null-no-default must NOT fire inside CREATE TABLE block; got %v", results)
	}
}

func TestLint_DropTableWithoutIfExists_Error(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "00001_drop_table.sql", "DROP TABLE foo;")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if !hasRule(results, "drop-table") {
		t.Fatalf("expected drop-table error, got %v", results)
	}
	r := results[0]
	if r.Severity != "error" {
		t.Fatalf("drop-table severity = %q, want %q", r.Severity, "error")
	}
}

func TestLint_DropTableWithIfExists_NoError(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "00001_drop_table_if.sql", "DROP TABLE IF EXISTS foo;")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if hasRule(results, "drop-table") {
		t.Fatalf("DROP TABLE IF EXISTS must not fire drop-table rule; got %v", results)
	}
}

func TestLint_UnboundedUpdate_Error(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "00001_update.sql", "UPDATE foo SET col = 1;")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if !hasRule(results, "unbounded-update") {
		t.Fatalf("expected unbounded-update error, got %v", results)
	}
	r := results[0]
	if r.Severity != "error" {
		t.Fatalf("unbounded-update severity = %q, want %q", r.Severity, "error")
	}
}

func TestLint_UpdateWithWhere_NoError(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "00001_update_where.sql", "UPDATE foo SET col = 1 WHERE id = 1;")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if hasRule(results, "unbounded-update") {
		t.Fatalf("UPDATE with WHERE must not fire unbounded-update; got %v", results)
	}
}

func TestLint_UnboundedDelete_Error(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "00001_delete.sql", "DELETE FROM foo;")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if !hasRule(results, "unbounded-delete") {
		t.Fatalf("expected unbounded-delete error, got %v", results)
	}
	r := results[0]
	if r.Severity != "error" {
		t.Fatalf("unbounded-delete severity = %q, want %q", r.Severity, "error")
	}
}

func TestLint_DeleteWithWhere_NoError(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "00001_delete_where.sql", "DELETE FROM foo WHERE id = 1;")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if hasRule(results, "unbounded-delete") {
		t.Fatalf("DELETE with WHERE must not fire unbounded-delete; got %v", results)
	}
}

func TestLint_DownSectionNotLinted(t *testing.T) {
	// Dangerous patterns only in the Down section must not produce results.
	dir := t.TempDir()
	writeDownOnlyMigration(t, dir, "00001_down_only.sql",
		"DROP TABLE foo;\nUPDATE foo SET col = 1;\nDELETE FROM foo;")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Down section must not be linted; got results: %v", results)
	}
}

func TestLint_MultipleFiles_ResultsHaveCorrectFilePath(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "00001_drop.sql", "DROP TABLE foo;")
	writeMigration(t, dir, "00002_update.sql", "UPDATE bar SET x = 1;")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (one per file), got %d: %v", len(results), results)
	}

	filesSeen := map[string]bool{}
	for _, r := range results {
		filesSeen[filepath.Base(r.File)] = true
	}
	if !filesSeen["00001_drop.sql"] {
		t.Fatalf("expected result for 00001_drop.sql, got file paths: %v", filesSeen)
	}
	if !filesSeen["00002_update.sql"] {
		t.Fatalf("expected result for 00002_update.sql, got file paths: %v", filesSeen)
	}
}

func TestLint_NonSQLFilesSkipped(t *testing.T) {
	dir := t.TempDir()
	// Write a .txt file with dangerous SQL content — must be skipped.
	txtPath := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(txtPath, []byte("-- +migrate Up\nDROP TABLE foo;\n"), 0o644); err != nil {
		t.Fatalf("write txt: %v", err)
	}

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("non-SQL files must be skipped; got results: %v", results)
	}
}

func TestLint_LintResultFields(t *testing.T) {
	// Verify all LintResult fields are populated correctly.
	dir := t.TempDir()
	path := writeMigration(t, dir, "00001_check.sql", "UPDATE foo SET col = 1;")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}

	r := results[0]
	if r.File != path {
		t.Fatalf("LintResult.File = %q, want %q", r.File, path)
	}
	if r.Line <= 0 {
		t.Fatalf("LintResult.Line = %d, want > 0", r.Line)
	}
	if r.Rule == "" {
		t.Fatal("LintResult.Rule must not be empty")
	}
	if r.Severity == "" {
		t.Fatal("LintResult.Severity must not be empty")
	}
	if r.Message == "" {
		t.Fatal("LintResult.Message must not be empty")
	}
}

func TestLint_NonExistentDir_Error(t *testing.T) {
	_, err := Lint("/does/not/exist/at/all")
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
}

func TestLint_DollarQuotedBlock_NoFalsePositive(t *testing.T) {
	// DDL patterns inside $$ bodies must not fire lint rules.
	dir := t.TempDir()
	writeMigration(t, dir, "00001_fn.sql",
		"-- +migrate StatementBegin\nCREATE OR REPLACE FUNCTION fn()\nAS $$\nBEGIN\n    UPDATE foo SET col = 1;\n    DELETE FROM bar;\nEND;\n$$;\n-- +migrate StatementEnd")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if hasRule(results, "unbounded-update") {
		t.Fatalf("unbounded-update must not fire inside $$ block; got %v", results)
	}
	if hasRule(results, "unbounded-delete") {
		t.Fatalf("unbounded-delete must not fire inside $$ block; got %v", results)
	}
}

func TestLint_DollarQuotedBlock_DDLOutsideStillCaught(t *testing.T) {
	// Dangerous DDL before or after a $$ block must still be caught.
	dir := t.TempDir()
	writeMigration(t, dir, "00001_mixed.sql",
		"UPDATE foo SET col = 1;\n-- +migrate StatementBegin\nCREATE OR REPLACE FUNCTION fn()\nAS $$\nBEGIN\n    RETURN NEW;\nEND;\n$$;\n-- +migrate StatementEnd")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if !hasRule(results, "unbounded-update") {
		t.Fatalf("unbounded-update must fire for UPDATE outside $$ block; got %v", results)
	}
}

func TestLint_HasErrors_TrueWhenErrorsPresent(t *testing.T) {
	results := []LintResult{
		{Severity: "warning", Rule: "drop-column"},
		{Severity: "error", Rule: "drop-table"},
	}
	if !HasErrors(results) {
		t.Fatal("HasErrors must return true when error-severity results exist")
	}
}

func TestLint_HasErrors_FalseWhenOnlyWarnings(t *testing.T) {
	results := []LintResult{
		{Severity: "warning", Rule: "drop-column"},
	}
	if HasErrors(results) {
		t.Fatal("HasErrors must return false when only warning-severity results exist")
	}
}

func TestLint_HasErrors_FalseWhenEmpty(t *testing.T) {
	if HasErrors(nil) {
		t.Fatal("HasErrors must return false for nil results")
	}
}

func TestLint_NotNullWithDefault_InAlterTable_NoWarning(t *testing.T) {
	// ALTER TABLE with NOT NULL AND DEFAULT must not fire the warning.
	dir := t.TempDir()
	writeMigration(t, dir, "00001_notnull_default.sql",
		"ALTER TABLE foo ADD COLUMN bar TEXT NOT NULL DEFAULT '';")

	results, err := Lint(dir)
	if err != nil {
		t.Fatalf("Lint: unexpected error: %v", err)
	}
	if hasRule(results, "not-null-no-default") {
		t.Fatalf("NOT NULL with DEFAULT must not fire not-null-no-default; got %v", results)
	}
}
