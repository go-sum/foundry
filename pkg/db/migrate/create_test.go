package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// minimalTableSQL returns minimal DDL for a named table with a single column.
func minimalTableSQL(tableName string) string {
	return "CREATE TABLE IF NOT EXISTS " + tableName + " (\n    id UUID PRIMARY KEY DEFAULT gen_random_uuid()\n);\n"
}

// ---- Create — first run (no snapshot exists) --------------------------------

func TestCreate_DiscreteSchema_CreatesOwnFile(t *testing.T) {
	dir := t.TempDir()

	cfg := CreateConfig{
		MigrationsDir: dir,
		Schemas: []SchemaInput{
			{
				Name:     "auth",
				SQL:      minimalTableSQL("users"),
				Priority: 0,
				Discrete: true,
			},
		},
	}

	result, err := Create(cfg, "initial")
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("Files: got %d, want 1 — %v", len(result.Files), result.Files)
	}

	// File name should encode the schema name, not the migration name.
	base := filepath.Base(result.Files[0])
	if !strings.Contains(base, "auth") {
		t.Errorf("file name %q should contain schema name %q", base, "auth")
	}
	if !strings.HasSuffix(base, ".sql") {
		t.Errorf("file name %q should end in .sql", base)
	}

	// Snapshot should be written to .schema/<name>.sql.
	snapshotPath := filepath.Join(dir, ".schema", "auth.sql")
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Errorf("snapshot file missing at %q: %v", snapshotPath, err)
	}
}

func TestCreate_NonDiscreteSchema_CombinedFile(t *testing.T) {
	dir := t.TempDir()

	cfg := CreateConfig{
		MigrationsDir: dir,
		Schemas: []SchemaInput{
			{
				Name:     "contact",
				SQL:      minimalTableSQL("contact_submissions"),
				Priority: 1,
				Discrete: false,
			},
			{
				Name:     "analytics",
				SQL:      minimalTableSQL("page_views"),
				Priority: 2,
				Discrete: false,
			},
		},
	}

	result, err := Create(cfg, "migrate_features")
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	// Two non-discrete schemas with changes → one combined file.
	if len(result.Files) != 1 {
		t.Fatalf("Files: got %d, want 1 (non-discrete schemas combine) — %v", len(result.Files), result.Files)
	}

	// Combined file should contain both table DDL statements.
	data, err := os.ReadFile(result.Files[0])
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "contact_submissions") {
		t.Errorf("combined file missing contact_submissions: %s", content)
	}
	if !strings.Contains(content, "page_views") {
		t.Errorf("combined file missing page_views: %s", content)
	}
}

func TestCreate_MixedSchemas_DiscreteAndNonDiscrete(t *testing.T) {
	dir := t.TempDir()

	cfg := CreateConfig{
		MigrationsDir: dir,
		Schemas: []SchemaInput{
			{Name: "auth", SQL: minimalTableSQL("users"), Priority: 0, Discrete: true},
			{Name: "app", SQL: minimalTableSQL("posts"), Priority: 1, Discrete: false},
		},
	}

	result, err := Create(cfg, "initial")
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	// One file for discrete auth, one file for non-discrete app.
	if len(result.Files) != 2 {
		t.Fatalf("Files: got %d, want 2 — %v", len(result.Files), result.Files)
	}
}

// ---- Create — second run on unchanged schemas (idempotent) ------------------

func TestCreate_UnchangedSchemas_NoFiles(t *testing.T) {
	dir := t.TempDir()

	cfg := CreateConfig{
		MigrationsDir: dir,
		Schemas: []SchemaInput{
			{Name: "auth", SQL: minimalTableSQL("users"), Priority: 0, Discrete: true},
		},
	}

	// First run.
	first, err := Create(cfg, "initial")
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if len(first.Files) != 1 {
		t.Fatalf("first Create: Files = %d, want 1", len(first.Files))
	}

	// Second run on same config — snapshot is now up to date.
	second, err := Create(cfg, "initial")
	if err != nil {
		t.Fatalf("second Create: %v", err)
	}
	if len(second.Files) != 0 {
		t.Errorf("second Create: Files = %d, want 0 (no changes detected)", len(second.Files))
	}
}

// ---- Create — incremental migration from schema change ----------------------

func TestCreate_IncrementalMigration_AddsColumn(t *testing.T) {
	dir := t.TempDir()

	// First run: table with only id column.
	sqlV1 := "CREATE TABLE IF NOT EXISTS widgets (\n    id UUID PRIMARY KEY DEFAULT gen_random_uuid()\n);\n"
	cfg := CreateConfig{
		MigrationsDir: dir,
		Schemas:       []SchemaInput{{Name: "widgets", SQL: sqlV1, Priority: 0, Discrete: false}},
	}
	if _, err := Create(cfg, "initial"); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	// Second run: add a name column.
	sqlV2 := "CREATE TABLE IF NOT EXISTS widgets (\n    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),\n    name TEXT NOT NULL\n);\n"
	cfg.Schemas[0].SQL = sqlV2

	second, err := Create(cfg, "add_name")
	if err != nil {
		t.Fatalf("second Create: %v", err)
	}
	if len(second.Files) != 1 {
		t.Fatalf("second Create: Files = %d, want 1 — %v", len(second.Files), second.Files)
	}

	data, err := os.ReadFile(second.Files[0])
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	// The incremental migration should only add the new column, not recreate the table.
	if !strings.Contains(content, "ADD COLUMN") {
		t.Errorf("incremental migration should contain ADD COLUMN: %s", content)
	}
	if strings.Count(content, "CREATE TABLE") > 0 {
		t.Errorf("incremental migration should not CREATE TABLE again: %s", content)
	}
}

// ---- Create — file format ---------------------------------------------------

func TestCreate_FileContainsMigrateMarkers(t *testing.T) {
	dir := t.TempDir()

	cfg := CreateConfig{
		MigrationsDir: dir,
		Schemas: []SchemaInput{
			{Name: "core", SQL: minimalTableSQL("things"), Priority: 0, Discrete: false},
		},
	}
	result, err := Create(cfg, "test_migration")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(result.Files) == 0 {
		t.Fatal("expected at least one file")
	}

	data, err := os.ReadFile(result.Files[0])
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "-- +migrate Up") {
		t.Errorf("file missing '-- +migrate Up' marker: %s", content)
	}
	if !strings.Contains(content, "-- +migrate Down") {
		t.Errorf("file missing '-- +migrate Down' marker: %s", content)
	}
}

func TestCreate_SequenceNumbers_Increment(t *testing.T) {
	dir := t.TempDir()

	// Write a pre-existing migration to force the sequence to start at 2.
	existingPath := filepath.Join(dir, "00001_existing.sql")
	if err := os.WriteFile(existingPath, []byte("-- +migrate Up\nSELECT 1;\n-- +migrate Down\nSELECT 1;\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg := CreateConfig{
		MigrationsDir: dir,
		Schemas: []SchemaInput{
			{Name: "new_schema", SQL: minimalTableSQL("new_table"), Priority: 0, Discrete: false},
		},
	}
	result, err := Create(cfg, "new_migration")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("Files: got %d, want 1", len(result.Files))
	}

	base := filepath.Base(result.Files[0])
	// The generated file should start with "00002_".
	if !strings.HasPrefix(base, "00002_") {
		t.Errorf("file name %q should start with 00002_ (sequence after existing 00001)", base)
	}
}

// ---- Create — empty result when no schemas have changes ----------------------

func TestCreate_NoChanges_EmptyResult(t *testing.T) {
	dir := t.TempDir()

	cfg := CreateConfig{
		MigrationsDir: dir,
		Schemas:       []SchemaInput{},
	}
	result, err := Create(cfg, "empty")
	if err != nil {
		t.Fatalf("Create with no schemas: %v", err)
	}
	if len(result.Files) != 0 {
		t.Errorf("expected no files for empty schema list, got %v", result.Files)
	}
}

// ---- Create — snapshot stored after first run --------------------------------

func TestCreate_SnapshotWrittenForEachSchema(t *testing.T) {
	dir := t.TempDir()

	cfg := CreateConfig{
		MigrationsDir: dir,
		Schemas: []SchemaInput{
			{Name: "auth", SQL: minimalTableSQL("users"), Priority: 0, Discrete: true},
			{Name: "app", SQL: minimalTableSQL("posts"), Priority: 1, Discrete: false},
		},
	}
	_, err := Create(cfg, "initial")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	for _, schemaName := range []string{"auth", "app"} {
		snapshotPath := filepath.Join(dir, ".schema", schemaName+".sql")
		data, err := os.ReadFile(snapshotPath)
		if err != nil {
			t.Errorf("snapshot file missing for schema %q: %v", schemaName, err)
			continue
		}
		// Snapshot content must equal the input SQL.
		for _, schema := range cfg.Schemas {
			if schema.Name == schemaName && string(data) != schema.SQL {
				t.Errorf("schema %q: snapshot content mismatch\ngot:  %q\nwant: %q",
					schemaName, string(data), schema.SQL)
			}
		}
	}
}

// ---- sanitizeName helper ----------------------------------------------------

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"initial_setup", "initial_setup"},
		{"My Feature", "my_feature"},
		{"add-users", "add_users"},
		{"pkg/auth/schema", "pkg_auth_schema"},
		{"UPPERCASE", "uppercase"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
