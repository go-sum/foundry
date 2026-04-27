package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/db/ddl"
)

// minimalTableSQL returns minimal DDL for a named table with a single column.
func minimalTableSQL(tableName string) string {
	return "CREATE TABLE IF NOT EXISTS " + tableName + " (\n    id UUID PRIMARY KEY DEFAULT gen_random_uuid()\n);\n"
}

// ---- Create — first run (no snapshot exists) --------------------------------

func TestCreate_GroupedSchema_CreatesOwnFile(t *testing.T) {
	dir := t.TempDir()

	cfg := CreateConfig{
		MigrationsDir: dir,
		BaseSchemas:   map[string]*ddl.Schema{},
		Schemas: []SchemaInput{
			{
				Name:     "auth",
				SQL:      minimalTableSQL("users"),
				Priority: 0,
				Group:    "auth",
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

	// File name should encode the group name (which equals the schema name here).
	base := filepath.Base(result.Files[0])
	if !strings.Contains(base, "auth") {
		t.Errorf("file name %q should contain group name %q", base, "auth")
	}
	if !strings.HasSuffix(base, ".sql") {
		t.Errorf("file name %q should end in .sql", base)
	}
}

func TestCreate_UngroupedSchemas_CombinedFile(t *testing.T) {
	dir := t.TempDir()

	cfg := CreateConfig{
		MigrationsDir: dir,
		BaseSchemas:   map[string]*ddl.Schema{},
		Schemas: []SchemaInput{
			{
				Name:     "contact",
				SQL:      minimalTableSQL("contact_submissions"),
				Priority: 1,
				Group:    "",
			},
			{
				Name:     "analytics",
				SQL:      minimalTableSQL("page_views"),
				Priority: 2,
				Group:    "",
			},
		},
	}

	result, err := Create(cfg, "migrate_features")
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	// Two ungrouped schemas with changes → one combined file.
	if len(result.Files) != 1 {
		t.Fatalf("Files: got %d, want 1 (ungrouped schemas combine) — %v", len(result.Files), result.Files)
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

func TestCreate_MixedGroupedAndUngrouped(t *testing.T) {
	dir := t.TempDir()

	cfg := CreateConfig{
		MigrationsDir: dir,
		BaseSchemas:   map[string]*ddl.Schema{},
		Schemas: []SchemaInput{
			{Name: "auth", SQL: minimalTableSQL("users"), Priority: 0, Group: "auth"},
			{Name: "app", SQL: minimalTableSQL("posts"), Priority: 1, Group: ""},
		},
	}

	result, err := Create(cfg, "initial")
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	// One file for grouped auth, one file for ungrouped app.
	if len(result.Files) != 2 {
		t.Fatalf("Files: got %d, want 2 — %v", len(result.Files), result.Files)
	}
}

func TestCreate_MultiSchemaGroup_MergedIntoOneFile(t *testing.T) {
	dir := t.TempDir()

	cfg := CreateConfig{
		MigrationsDir: dir,
		BaseSchemas:   map[string]*ddl.Schema{},
		Schemas: []SchemaInput{
			{Name: "auth", SQL: minimalTableSQL("users"), Priority: 60, Group: "auth"},
			{Name: "auth_provider", SQL: minimalTableSQL("oauth_clients"), Priority: 70, Group: "auth"},
		},
	}

	result, err := Create(cfg, "initial")
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	// Two schemas in the same group → one file.
	if len(result.Files) != 1 {
		t.Fatalf("Files: got %d, want 1 (same-group schemas merge) — %v", len(result.Files), result.Files)
	}

	// File name should contain the group name.
	base := filepath.Base(result.Files[0])
	if !strings.Contains(base, "auth") {
		t.Errorf("file name %q should contain group name %q", base, "auth")
	}

	// Combined file should contain both tables.
	data, err := os.ReadFile(result.Files[0])
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "users") {
		t.Errorf("merged file missing users table: %s", content)
	}
	if !strings.Contains(content, "oauth_clients") {
		t.Errorf("merged file missing oauth_clients table: %s", content)
	}
}

func TestCreate_GroupOrdering_ByMaxPriority(t *testing.T) {
	dir := t.TempDir()

	// queue (50) < auth group max (70) < ungrouped (100)
	cfg := CreateConfig{
		MigrationsDir: dir,
		BaseSchemas:   map[string]*ddl.Schema{},
		Schemas: []SchemaInput{
			{Name: "auth", SQL: minimalTableSQL("users"), Priority: 60, Group: "auth"},
			{Name: "auth_provider", SQL: minimalTableSQL("oauth_clients"), Priority: 70, Group: "auth"},
			{Name: "queue", SQL: minimalTableSQL("jobs"), Priority: 50, Group: "queue"},
			{Name: "contact", SQL: minimalTableSQL("contact_submissions"), Priority: 100, Group: ""},
		},
	}

	result, err := Create(cfg, "app")
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	// queue group, auth group, ungrouped → 3 files
	if len(result.Files) != 3 {
		t.Fatalf("Files: got %d, want 3 — %v", len(result.Files), result.Files)
	}

	// Files should be in priority order: queue (00001), auth (00002), ungrouped (00003).
	if !strings.Contains(filepath.Base(result.Files[0]), "queue") {
		t.Errorf("first file should be queue group, got %q", filepath.Base(result.Files[0]))
	}
	if !strings.Contains(filepath.Base(result.Files[1]), "auth") {
		t.Errorf("second file should be auth group, got %q", filepath.Base(result.Files[1]))
	}
	// Third file is ungrouped, named after the name param "app".
	if !strings.Contains(filepath.Base(result.Files[2]), "app") {
		t.Errorf("third file should use migration name 'app', got %q", filepath.Base(result.Files[2]))
	}
}

// ---- Create — second run on unchanged schemas (idempotent) ------------------

func TestCreate_UnchangedSchemas_NoFiles(t *testing.T) {
	dir := t.TempDir()

	sql := minimalTableSQL("users")
	cfg := CreateConfig{
		MigrationsDir: dir,
		BaseSchemas:   map[string]*ddl.Schema{"auth": ddl.Parse(sql)},
		Schemas: []SchemaInput{
			{Name: "auth", SQL: sql, Priority: 0, Group: "auth"},
		},
	}

	// Single pass with BaseSchemas already matching current SQL → no changes.
	result, err := Create(cfg, "initial")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(result.Files) != 0 {
		t.Errorf("Create: Files = %d, want 0 (no changes detected)", len(result.Files))
	}
}

// ---- Create — incremental migration from schema change ----------------------

func TestCreate_IncrementalMigration_AddsColumn(t *testing.T) {
	dir := t.TempDir()

	sqlV1 := "CREATE TABLE IF NOT EXISTS widgets (\n    id UUID PRIMARY KEY DEFAULT gen_random_uuid()\n);\n"
	sqlV2 := "CREATE TABLE IF NOT EXISTS widgets (\n    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),\n    name TEXT NOT NULL\n);\n"

	cfg := CreateConfig{
		MigrationsDir: dir,
		BaseSchemas:   map[string]*ddl.Schema{"widgets": ddl.Parse(sqlV1)},
		Schemas:       []SchemaInput{{Name: "widgets", SQL: sqlV2, Priority: 0, Group: ""}},
	}

	result, err := Create(cfg, "add_name")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("Create: Files = %d, want 1 — %v", len(result.Files), result.Files)
	}

	data, err := os.ReadFile(result.Files[0])
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
		BaseSchemas:   map[string]*ddl.Schema{},
		Schemas: []SchemaInput{
			{Name: "core", SQL: minimalTableSQL("things"), Priority: 0, Group: ""},
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
	if !strings.HasPrefix(content, "-- Auto-generated migration - do not edit") {
		t.Errorf("file missing auto-generated header: %s", content)
	}
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
		BaseSchemas:   map[string]*ddl.Schema{},
		Schemas: []SchemaInput{
			{Name: "new_schema", SQL: minimalTableSQL("new_table"), Priority: 0, Group: ""},
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
		BaseSchemas:   map[string]*ddl.Schema{},
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

// ---- Create — nil BaseSchemas returns error ----------------------------------

func TestCreate_NilBaseSchemas_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	cfg := CreateConfig{
		MigrationsDir: dir,
		Schemas:       []SchemaInput{{Name: "x", SQL: minimalTableSQL("x"), Priority: 0}},
	}
	_, err := Create(cfg, "test")
	if err == nil {
		t.Fatal("expected error for nil BaseSchemas")
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
