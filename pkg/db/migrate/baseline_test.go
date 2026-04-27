package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/db/ddl"
)

// writeBaselineMigration writes a full up/down migration file to dir using a
// numeric sequence number as the filename prefix (e.g. 00001_name.sql).
func writeBaselineMigration(t *testing.T, dir string, seq int, name, upSQL, downSQL string) string {
	t.Helper()
	prefix := seqPrefix(seq)
	fileName := filepath.Join(dir, prefix+"_"+name+".sql")
	content := "-- Auto-generated migration - do not edit\n\n-- +migrate Up\n" + upSQL + "\n\n-- +migrate Down\n" + downSQL + "\n"
	if err := os.WriteFile(fileName, []byte(content), 0o644); err != nil {
		t.Fatalf("writeBaselineMigration: WriteFile %s: %v", fileName, err)
	}
	return fileName
}

// seqPrefix formats an integer as a zero-padded 5-digit string (e.g. 1 → "00001").
func seqPrefix(n int) string {
	s := ""
	for i := 0; i < 5; i++ {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// ---- BuildBaseline — empty / nonexistent directory ---------------------------

func TestBuildBaseline_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	schemas := []SchemaInput{
		{Name: "auth", SQL: minimalTableSQL("users"), Priority: 0},
	}

	result, err := BuildBaseline(dir, schemas)
	if err != nil {
		t.Fatalf("BuildBaseline on empty dir: unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("BuildBaseline returned nil map")
	}
	if len(result) != 1 {
		t.Fatalf("result length: got %d, want 1", len(result))
	}
	s, ok := result["auth"]
	if !ok {
		t.Fatal("result missing key 'auth'")
	}
	// Empty migrations directory → baseline is an empty schema (no tables).
	if len(s.Tables) != 0 {
		t.Errorf("Tables: got %d, want 0 for empty migrations dir", len(s.Tables))
	}
}

func TestBuildBaseline_NonExistentDir(t *testing.T) {
	// Use a path guaranteed not to exist.
	dir := filepath.Join(t.TempDir(), "does_not_exist")
	schemas := []SchemaInput{
		{Name: "auth", SQL: minimalTableSQL("users"), Priority: 0},
	}

	result, err := BuildBaseline(dir, schemas)
	if err != nil {
		t.Fatalf("BuildBaseline on nonexistent dir: unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("BuildBaseline returned nil for nonexistent dir")
	}
	s, ok := result["auth"]
	if !ok {
		t.Fatal("result missing key 'auth'")
	}
	if len(s.Tables) != 0 {
		t.Errorf("Tables: got %d, want 0 for nonexistent dir", len(s.Tables))
	}
}

// ---- BuildBaseline — single migration ----------------------------------------

func TestBuildBaseline_SingleMigration(t *testing.T) {
	dir := t.TempDir()
	createSQL := minimalTableSQL("users")
	writeBaselineMigration(t, dir, 1, "create_users", createSQL, "DROP TABLE IF EXISTS users CASCADE;")

	schemas := []SchemaInput{
		{Name: "auth", SQL: createSQL, Priority: 0},
	}

	result, err := BuildBaseline(dir, schemas)
	if err != nil {
		t.Fatalf("BuildBaseline: %v", err)
	}

	s, ok := result["auth"]
	if !ok {
		t.Fatal("result missing key 'auth'")
	}
	if len(s.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(s.Tables))
	}
	if s.Tables[0].Name != "users" {
		t.Errorf("Table name: got %q, want %q", s.Tables[0].Name, "users")
	}
}

// ---- BuildBaseline — sequential migrations build up schema -------------------

func TestBuildBaseline_MultiMigration_AddsColumn(t *testing.T) {
	dir := t.TempDir()

	// Migration 1: create users table.
	writeBaselineMigration(t, dir, 1, "create_users",
		minimalTableSQL("users"),
		"DROP TABLE IF EXISTS users CASCADE;",
	)
	// Migration 2: add email column.
	writeBaselineMigration(t, dir, 2, "add_email",
		"ALTER TABLE users ADD COLUMN email TEXT NOT NULL;",
		"ALTER TABLE users DROP COLUMN IF EXISTS email;",
	)

	schemaSQL := "CREATE TABLE IF NOT EXISTS users (\n    id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),\n    email TEXT NOT NULL\n);\n"
	schemas := []SchemaInput{
		{Name: "auth", SQL: schemaSQL, Priority: 0},
	}

	result, err := BuildBaseline(dir, schemas)
	if err != nil {
		t.Fatalf("BuildBaseline: %v", err)
	}

	s, ok := result["auth"]
	if !ok {
		t.Fatal("result missing key 'auth'")
	}
	if len(s.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(s.Tables))
	}
	table := s.Tables[0]
	if table.Name != "users" {
		t.Errorf("Table name: got %q, want %q", table.Name, "users")
	}
	cols := make(map[string]bool, len(table.Columns))
	for _, c := range table.Columns {
		cols[c.Name] = true
	}
	if !cols["id"] {
		t.Error("column 'id' missing after two-migration replay")
	}
	if !cols["email"] {
		t.Error("column 'email' missing — second migration was not applied")
	}
}

// ---- BuildBaseline — idempotent after Create ---------------------------------

func TestBuildBaseline_IdempotentAfterCreate(t *testing.T) {
	dir := t.TempDir()

	schemaSQL := minimalTableSQL("orders")
	schemas := []SchemaInput{
		{Name: "app", SQL: schemaSQL, Priority: 0},
	}

	// First Create: baseline is empty → migration created.
	cfg1 := CreateConfig{
		MigrationsDir: dir,
		BaseSchemas:   map[string]*ddl.Schema{},
		Schemas:       schemas,
	}
	result1, err := Create(cfg1, "initial")
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if len(result1.Files) != 1 {
		t.Fatalf("first Create: Files = %d, want 1", len(result1.Files))
	}

	// Build baseline from the migration directory.
	baseline, err := BuildBaseline(dir, schemas)
	if err != nil {
		t.Fatalf("BuildBaseline: %v", err)
	}

	// Second Create: baseline matches current SQL → no changes.
	cfg2 := CreateConfig{
		MigrationsDir: dir,
		BaseSchemas:   baseline,
		Schemas:       schemas,
	}
	result2, err := Create(cfg2, "second_run")
	if err != nil {
		t.Fatalf("second Create: %v", err)
	}
	if len(result2.Files) != 0 {
		t.Errorf("second Create produced %d file(s); want 0 — baseline replay must be idempotent", len(result2.Files))
	}
}

// ---- BuildBaseline — multiple schema inputs partitioned per-schema -----------

func TestBuildBaseline_MultipleSchemaInputs(t *testing.T) {
	dir := t.TempDir()

	// Migration contains tables for two different schemas.
	writeBaselineMigration(t, dir, 1, "create_all",
		minimalTableSQL("users")+minimalTableSQL("orders"),
		"DROP TABLE IF EXISTS users CASCADE;\nDROP TABLE IF EXISTS orders CASCADE;",
	)

	authSQL := minimalTableSQL("users")
	appSQL := minimalTableSQL("orders")
	schemas := []SchemaInput{
		{Name: "auth", SQL: authSQL, Priority: 0},
		{Name: "app", SQL: appSQL, Priority: 1},
	}

	result, err := BuildBaseline(dir, schemas)
	if err != nil {
		t.Fatalf("BuildBaseline: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("result length: got %d, want 2", len(result))
	}

	authSchema, ok := result["auth"]
	if !ok {
		t.Fatal("result missing key 'auth'")
	}
	appSchema, ok := result["app"]
	if !ok {
		t.Fatal("result missing key 'app'")
	}

	// auth schema should contain only users.
	if len(authSchema.Tables) != 1 {
		t.Fatalf("auth Tables: got %d, want 1", len(authSchema.Tables))
	}
	if authSchema.Tables[0].Name != "users" {
		t.Errorf("auth table name: got %q, want %q", authSchema.Tables[0].Name, "users")
	}

	// app schema should contain only orders.
	if len(appSchema.Tables) != 1 {
		t.Fatalf("app Tables: got %d, want 1", len(appSchema.Tables))
	}
	if appSchema.Tables[0].Name != "orders" {
		t.Errorf("app table name: got %q, want %q", appSchema.Tables[0].Name, "orders")
	}
}

// ---- BuildBaseline — Down section must NOT be applied during replay ----------

func TestBuildBaseline_MigrationWithDownSection_IgnoresDown(t *testing.T) {
	dir := t.TempDir()

	// Up creates users; Down would drop it.
	// BuildBaseline must replay only Up, leaving users in the result.
	writeBaselineMigration(t, dir, 1, "create_users",
		minimalTableSQL("users"),
		"DROP TABLE IF EXISTS users CASCADE;",
	)

	schemas := []SchemaInput{
		{Name: "auth", SQL: minimalTableSQL("users"), Priority: 0},
	}

	result, err := BuildBaseline(dir, schemas)
	if err != nil {
		t.Fatalf("BuildBaseline: %v", err)
	}

	s, ok := result["auth"]
	if !ok {
		t.Fatal("result missing key 'auth'")
	}
	if len(s.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1 — Down SQL must not be applied", len(s.Tables))
	}
	if s.Tables[0].Name != "users" {
		t.Errorf("Table name: got %q, want %q", s.Tables[0].Name, "users")
	}

	// Confirm the Down SQL would actually remove the table if applied (sanity check).
	withDown := ddl.Apply(s, "DROP TABLE IF EXISTS users CASCADE;")
	if len(withDown.Tables) != 0 {
		t.Errorf("sanity: applying Down SQL should drop the table; got %d tables", len(withDown.Tables))
	}
	// But the baseline itself (before Down) must still hold the table.
	if len(s.Tables) != 1 {
		t.Errorf("original baseline s was mutated — Apply must not mutate its input")
	}
}

// ---- BuildBaseline — migrations replayed in version order --------------------

func TestBuildBaseline_OrderedReplay(t *testing.T) {
	dir := t.TempDir()

	// Write in reverse filename order to test that version ordering is enforced.
	// Migration 2: drop the column that migration 1 would add.
	// If replay order is wrong, the ADD COLUMN will run after DROP and we'll have "extra".
	// Correct order (1 then 2): create table → add extra → drop extra → extra absent.
	writeBaselineMigration(t, dir, 2, "drop_extra",
		"ALTER TABLE widgets DROP COLUMN IF EXISTS extra;",
		"ALTER TABLE widgets ADD COLUMN extra TEXT;",
	)
	writeBaselineMigration(t, dir, 1, "create_widgets",
		"CREATE TABLE IF NOT EXISTS widgets (\n    id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),\n    extra TEXT\n);\n",
		"DROP TABLE IF EXISTS widgets CASCADE;",
	)

	schemaSQL := minimalTableSQL("widgets") // only id; extra is dropped by migration 2
	schemas := []SchemaInput{
		{Name: "app", SQL: schemaSQL, Priority: 0},
	}

	result, err := BuildBaseline(dir, schemas)
	if err != nil {
		t.Fatalf("BuildBaseline: %v", err)
	}

	s, ok := result["app"]
	if !ok {
		t.Fatal("result missing key 'app'")
	}
	if len(s.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(s.Tables))
	}
	for _, c := range s.Tables[0].Columns {
		if c.Name == "extra" {
			t.Errorf("column 'extra' present — replay order is wrong (migration 2 should run after migration 1)")
		}
	}
	found := false
	for _, c := range s.Tables[0].Columns {
		if c.Name == "id" {
			found = true
		}
	}
	if !found {
		t.Error("column 'id' missing after ordered replay")
	}
}

// ---- helper: table-driven suite for no-files / no-schemas edge cases ---------

func TestBuildBaseline_NoSchemaInputs(t *testing.T) {
	dir := t.TempDir()
	writeBaselineMigration(t, dir, 1, "create_users", minimalTableSQL("users"), "DROP TABLE IF EXISTS users CASCADE;")

	// An empty schemas slice means no schema inputs to partition by → empty map.
	result, err := BuildBaseline(dir, nil)
	if err != nil {
		t.Fatalf("BuildBaseline with nil schemas: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("result length: got %d, want 0 for nil schemas input", len(result))
	}
}

// ---- helper: verify Down SQL content in migration files ----------------------

func TestBuildBaseline_DownSQLNotEmpty(t *testing.T) {
	// This test is about the migration file format, not BuildBaseline itself.
	// It confirms that writeMigration embeds Down SQL correctly so that
	// the ParseDir + ParseFile path under BuildBaseline reads the right sections.
	dir := t.TempDir()
	writeBaselineMigration(t, dir, 1, "test", "SELECT 1;", "SELECT 2;")

	migs, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if len(migs) != 1 {
		t.Fatalf("migrations: got %d, want 1", len(migs))
	}
	if !strings.Contains(migs[0].UpSQL, "SELECT 1") {
		t.Errorf("UpSQL = %q, want to contain 'SELECT 1'", migs[0].UpSQL)
	}
	if !strings.Contains(migs[0].DownSQL, "SELECT 2") {
		t.Errorf("DownSQL = %q, want to contain 'SELECT 2'", migs[0].DownSQL)
	}
}
