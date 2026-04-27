package ddl

import (
	"testing"
)

// minimalTableSQL returns minimal DDL for a named table with a single UUID PK column.
func minimalTableSQL(tableName string) string {
	return "CREATE TABLE IF NOT EXISTS " + tableName + " (\n    id UUID PRIMARY KEY DEFAULT gen_random_uuid()\n);\n"
}

func TestApply_CreateTable_EmptyBase(t *testing.T) {
	sql := "CREATE TABLE IF NOT EXISTS users (\n    id UUID PRIMARY KEY DEFAULT gen_random_uuid()\n);\n"
	result := Apply(&Schema{}, sql)
	if len(result.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(result.Tables))
	}
	if result.Tables[0].Name != "users" {
		t.Errorf("Table name: got %q, want %q", result.Tables[0].Name, "users")
	}
}

func TestApply_AlterAddColumn(t *testing.T) {
	base := Parse("CREATE TABLE IF NOT EXISTS widgets (\n    id UUID PRIMARY KEY DEFAULT gen_random_uuid()\n);\n")
	sql := "ALTER TABLE widgets ADD COLUMN name TEXT NOT NULL;"
	result := Apply(base, sql)
	if len(result.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(result.Tables))
	}
	cols := colsByName(result.Tables[0].Columns)
	if !cols["id"] {
		t.Error("column 'id' missing after ALTER")
	}
	if !cols["name"] {
		t.Error("column 'name' not added by ALTER")
	}
}

func TestApply_AlterDropColumn(t *testing.T) {
	base := Parse("CREATE TABLE IF NOT EXISTS widgets (\n    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),\n    name TEXT NOT NULL\n);\n")
	sql := "ALTER TABLE widgets DROP COLUMN IF EXISTS name;"
	result := Apply(base, sql)
	if len(result.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(result.Tables))
	}
	cols := colsByName(result.Tables[0].Columns)
	if cols["name"] {
		t.Error("column 'name' should have been dropped")
	}
	if !cols["id"] {
		t.Error("column 'id' should still be present")
	}
}

func TestApply_DropTable(t *testing.T) {
	base := Parse("CREATE TABLE IF NOT EXISTS users (\n    id UUID PRIMARY KEY DEFAULT gen_random_uuid()\n);\nCREATE TABLE IF NOT EXISTS posts (\n    id UUID PRIMARY KEY DEFAULT gen_random_uuid()\n);\n")
	sql := "DROP TABLE IF EXISTS users CASCADE;"
	result := Apply(base, sql)
	tables := tablesByName(result.Tables)
	if _, ok := tables["users"]; ok {
		t.Error("table 'users' should have been dropped")
	}
	if _, ok := tables["posts"]; !ok {
		t.Error("table 'posts' should still be present")
	}
}

func TestApply_CreateAndDropIndex(t *testing.T) {
	// Create index.
	base := &Schema{}
	sqlCreate := "CREATE TABLE IF NOT EXISTS users (\n    id UUID PRIMARY KEY DEFAULT gen_random_uuid()\n);\nCREATE INDEX idx_users_id ON users (id);\n"
	withIndex := Apply(base, sqlCreate)
	if len(withIndex.Indexes) != 1 {
		t.Fatalf("after create: Indexes = %d, want 1", len(withIndex.Indexes))
	}

	// Drop index.
	sqlDrop := "DROP INDEX IF EXISTS idx_users_id;"
	noIndex := Apply(withIndex, sqlDrop)
	if len(noIndex.Indexes) != 0 {
		t.Errorf("after drop: Indexes = %d, want 0", len(noIndex.Indexes))
	}
}

func TestApply_CreateAndDropExtension(t *testing.T) {
	base := &Schema{}
	withExt := Apply(base, "CREATE EXTENSION IF NOT EXISTS pgcrypto;")
	if len(withExt.Extensions) != 1 || withExt.Extensions[0] != "pgcrypto" {
		t.Fatalf("after create: Extensions = %v, want [pgcrypto]", withExt.Extensions)
	}

	noExt := Apply(withExt, "DROP EXTENSION IF EXISTS pgcrypto;")
	if len(noExt.Extensions) != 0 {
		t.Errorf("after drop: Extensions = %v, want []", noExt.Extensions)
	}
}

func TestApply_CreateAndDropFunction(t *testing.T) {
	fnSQL := "CREATE OR REPLACE FUNCTION set_updated_at()\nRETURNS TRIGGER AS $$\nBEGIN\n  NEW.updated_at = now();\n  RETURN NEW;\nEND;\n$$ LANGUAGE plpgsql;"
	withFn := Apply(&Schema{}, fnSQL)
	if len(withFn.Functions) != 1 {
		t.Fatalf("after create: Functions = %d, want 1", len(withFn.Functions))
	}
	if withFn.Functions[0].Name != "set_updated_at" {
		t.Errorf("Function name: got %q, want %q", withFn.Functions[0].Name, "set_updated_at")
	}

	noFn := Apply(withFn, "DROP FUNCTION IF EXISTS set_updated_at;")
	if len(noFn.Functions) != 0 {
		t.Errorf("after drop: Functions = %d, want 0", len(noFn.Functions))
	}
}

func TestApply_DropTrigger(t *testing.T) {
	trigSQL := `DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_users_updated_at') THEN
    CREATE TRIGGER trg_users_updated_at
      BEFORE UPDATE ON users
      FOR EACH ROW EXECUTE FUNCTION set_updated_at();
  END IF;
END $$;`
	withTrig := Apply(&Schema{}, trigSQL)
	if len(withTrig.Triggers) != 1 {
		t.Fatalf("after create: Triggers = %d, want 1", len(withTrig.Triggers))
	}

	noTrig := Apply(withTrig, "DROP TRIGGER IF EXISTS trg_users_updated_at ON users;")
	if len(noTrig.Triggers) != 0 {
		t.Errorf("after drop: Triggers = %d, want 0", len(noTrig.Triggers))
	}
}

func TestApply_UnrecognizedSQL_Skipped(t *testing.T) {
	base := Parse("CREATE TABLE IF NOT EXISTS users (\n    id UUID PRIMARY KEY DEFAULT gen_random_uuid()\n);\n")
	sql := "SELECT 1; COMMENT ON TABLE users IS 'test'; SET search_path TO public;"
	result := Apply(base, sql)
	// Schema should be unchanged.
	if len(result.Tables) != 1 {
		t.Errorf("Tables: got %d, want 1 (unrecognized SQL should be skipped)", len(result.Tables))
	}
	if result.Tables[0].Name != "users" {
		t.Errorf("Table name changed unexpectedly: %q", result.Tables[0].Name)
	}
}

func TestApply_DoesNotMutateBase(t *testing.T) {
	base := Parse("CREATE TABLE IF NOT EXISTS users (\n    id UUID PRIMARY KEY DEFAULT gen_random_uuid()\n);\n")
	originalTableCount := len(base.Tables)
	originalColCount := len(base.Tables[0].Columns)

	Apply(base, "ALTER TABLE users ADD COLUMN name TEXT NOT NULL;")

	if len(base.Tables) != originalTableCount {
		t.Errorf("base.Tables mutated: got %d, want %d", len(base.Tables), originalTableCount)
	}
	if len(base.Tables[0].Columns) != originalColCount {
		t.Errorf("base.Tables[0].Columns mutated: got %d, want %d", len(base.Tables[0].Columns), originalColCount)
	}
}

func TestApply_RoundTrip_DiffThenApply(t *testing.T) {
	sqlA := "CREATE TABLE IF NOT EXISTS widgets (\n    id UUID PRIMARY KEY DEFAULT gen_random_uuid()\n);\n"
	sqlB := "CREATE TABLE IF NOT EXISTS widgets (\n    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),\n    name TEXT NOT NULL\n);\n"

	schemaA := Parse(sqlA)
	schemaB := Parse(sqlB)

	diff := Diff(schemaA, schemaB)
	if diff.Empty {
		t.Fatal("Diff should not be empty")
	}

	applied := Apply(schemaA, diff.UpSQL)
	colsApplied := colsByName(applied.Tables[0].Columns)
	colsB := colsByName(schemaB.Tables[0].Columns)

	for name := range colsB {
		if !colsApplied[name] {
			t.Errorf("column %q present in B but missing after Apply(A, Diff(A,B).UpSQL)", name)
		}
	}
}

// ---- Nil / empty inputs -------------------------------------------------------

func TestApply_NilBase(t *testing.T) {
	sql := minimalTableSQL("users")
	result := Apply(nil, sql)
	if result == nil {
		t.Fatal("Apply(nil, sql) returned nil; want non-nil schema")
	}
	if len(result.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(result.Tables))
	}
	if result.Tables[0].Name != "users" {
		t.Errorf("Table name: got %q, want %q", result.Tables[0].Name, "users")
	}
}

func TestApply_EmptySQL(t *testing.T) {
	base := Parse(minimalTableSQL("users"))
	result := Apply(base, "")
	if len(result.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(result.Tables))
	}
	if result.Tables[0].Name != "users" {
		t.Errorf("Table name mutated: got %q, want %q", result.Tables[0].Name, "users")
	}
	if len(result.Tables[0].Columns) != len(base.Tables[0].Columns) {
		t.Errorf("Columns mutated: got %d, want %d", len(result.Tables[0].Columns), len(base.Tables[0].Columns))
	}
}

func TestApply_WhitespaceAndCommentsOnly(t *testing.T) {
	base := Parse(minimalTableSQL("users"))
	sql := "-- this is a comment\n   \n-- another comment\n"
	result := Apply(base, sql)
	if len(result.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(result.Tables))
	}
	if len(result.Tables[0].Columns) != len(base.Tables[0].Columns) {
		t.Errorf("Columns changed unexpectedly after whitespace/comment-only SQL")
	}
}

// ---- Idempotency / duplicate operations ---------------------------------------

func TestApply_AddColumn_Duplicate(t *testing.T) {
	base := Parse(minimalTableSQL("widgets"))
	// Apply the same ADD COLUMN twice.
	sql := "ALTER TABLE widgets ADD COLUMN name TEXT NOT NULL;"
	after1 := Apply(base, sql)
	after2 := Apply(after1, sql)

	if len(after2.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(after2.Tables))
	}
	wantCols := 2 // id + name
	if len(after2.Tables[0].Columns) != wantCols {
		t.Errorf("Columns: got %d, want %d (duplicate ADD COLUMN must not duplicate)", len(after2.Tables[0].Columns), wantCols)
	}
}

func TestApply_CreateTable_AlreadyExists(t *testing.T) {
	base := Parse(minimalTableSQL("users"))
	// Apply the same CREATE TABLE again.
	result := Apply(base, minimalTableSQL("users"))
	if len(result.Tables) != 1 {
		t.Errorf("Tables: got %d, want 1 (duplicate CREATE TABLE must not duplicate)", len(result.Tables))
	}
}

func TestApply_CreateExtension_Duplicate(t *testing.T) {
	base := Apply(&Schema{}, "CREATE EXTENSION IF NOT EXISTS pgcrypto;")
	result := Apply(base, "CREATE EXTENSION IF NOT EXISTS pgcrypto;")
	if len(result.Extensions) != 1 {
		t.Errorf("Extensions: got %d, want 1 (duplicate must not add twice)", len(result.Extensions))
	}
	if result.Extensions[0] != "pgcrypto" {
		t.Errorf("Extension name: got %q, want %q", result.Extensions[0], "pgcrypto")
	}
}

func TestApply_CreateIndex_Duplicate(t *testing.T) {
	createSQL := minimalTableSQL("users") + "CREATE INDEX idx_users_id ON users (id);\n"
	base := Apply(&Schema{}, createSQL)
	if len(base.Indexes) != 1 {
		t.Fatalf("precondition: Indexes = %d, want 1", len(base.Indexes))
	}
	result := Apply(base, "CREATE INDEX idx_users_id ON users (id);")
	if len(result.Indexes) != 1 {
		t.Errorf("Indexes: got %d, want 1 (duplicate CREATE INDEX must not duplicate)", len(result.Indexes))
	}
}

// ---- Graceful no-ops on non-existent targets ----------------------------------

func TestApply_DropTable_NonExistent(t *testing.T) {
	base := Parse(minimalTableSQL("users"))
	result := Apply(base, "DROP TABLE IF EXISTS nonexistent CASCADE;")
	if len(result.Tables) != 1 {
		t.Errorf("Tables: got %d, want 1 (drop of nonexistent table must be no-op)", len(result.Tables))
	}
	if result.Tables[0].Name != "users" {
		t.Errorf("Table name changed: got %q, want %q", result.Tables[0].Name, "users")
	}
}

func TestApply_DropColumn_NonExistent(t *testing.T) {
	base := Parse(minimalTableSQL("users"))
	originalColCount := len(base.Tables[0].Columns)
	result := Apply(base, "ALTER TABLE users DROP COLUMN IF EXISTS ghost;")
	if len(result.Tables[0].Columns) != originalColCount {
		t.Errorf("Columns: got %d, want %d (drop of nonexistent column must be no-op)", len(result.Tables[0].Columns), originalColCount)
	}
}

func TestApply_DropColumn_TableNotFound(t *testing.T) {
	base := Parse(minimalTableSQL("users"))
	result := Apply(base, "ALTER TABLE missing_table DROP COLUMN IF EXISTS id;")
	// The "users" table must be completely unaffected.
	if len(result.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(result.Tables))
	}
	if result.Tables[0].Name != "users" {
		t.Errorf("Table name changed: got %q", result.Tables[0].Name)
	}
	if len(result.Tables[0].Columns) != len(base.Tables[0].Columns) {
		t.Errorf("Columns mutated: got %d, want %d", len(result.Tables[0].Columns), len(base.Tables[0].Columns))
	}
}

func TestApply_DropIndex_NonExistent(t *testing.T) {
	base := &Schema{}
	result := Apply(base, "DROP INDEX IF EXISTS idx_nonexistent;")
	if len(result.Indexes) != 0 {
		t.Errorf("Indexes: got %d, want 0", len(result.Indexes))
	}
}

func TestApply_DropExtension_NonExistent(t *testing.T) {
	base := &Schema{}
	result := Apply(base, "DROP EXTENSION IF EXISTS pgcrypto;")
	if len(result.Extensions) != 0 {
		t.Errorf("Extensions: got %d, want 0", len(result.Extensions))
	}
}

func TestApply_DropFunction_NonExistent(t *testing.T) {
	base := &Schema{}
	result := Apply(base, "DROP FUNCTION IF EXISTS set_updated_at;")
	if len(result.Functions) != 0 {
		t.Errorf("Functions: got %d, want 0", len(result.Functions))
	}
}

func TestApply_DropTrigger_NonExistent(t *testing.T) {
	base := &Schema{}
	result := Apply(base, "DROP TRIGGER IF EXISTS trg_users_updated_at ON users;")
	if len(result.Triggers) != 0 {
		t.Errorf("Triggers: got %d, want 0", len(result.Triggers))
	}
}

func TestApply_AddColumn_TableNotFound(t *testing.T) {
	base := Parse(minimalTableSQL("users"))
	result := Apply(base, "ALTER TABLE missing_table ADD COLUMN foo TEXT;")
	// No new table must be created; users table untouched.
	if len(result.Tables) != 1 {
		t.Errorf("Tables: got %d, want 1 (no table should be created for ADD COLUMN on unknown table)", len(result.Tables))
	}
	if result.Tables[0].Name != "users" {
		t.Errorf("Unexpected table name: %q", result.Tables[0].Name)
	}
}

// ---- Schema prefix handling ---------------------------------------------------

func TestApply_DropTable_WithSchemaPrefix(t *testing.T) {
	base := Parse(minimalTableSQL("users"))
	result := Apply(base, "DROP TABLE IF EXISTS public.users CASCADE;")
	if len(result.Tables) != 0 {
		t.Errorf("Tables: got %d, want 0 (schema-prefixed DROP TABLE should strip prefix)", len(result.Tables))
	}
}

func TestApply_AlterTable_WithSchemaPrefix(t *testing.T) {
	base := Parse(minimalTableSQL("users"))
	result := Apply(base, "ALTER TABLE public.users ADD COLUMN email TEXT NOT NULL;")
	if len(result.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(result.Tables))
	}
	cols := colsByName(result.Tables[0].Columns)
	if !cols["email"] {
		t.Error("column 'email' not added by schema-prefixed ALTER TABLE")
	}
}

// ---- Function and trigger replacement ----------------------------------------

func TestApply_ReplaceFunction(t *testing.T) {
	fnV1 := "CREATE OR REPLACE FUNCTION set_updated_at()\nRETURNS TRIGGER AS $$\nBEGIN\n  NEW.updated_at = now();\n  RETURN NEW;\nEND;\n$$ LANGUAGE plpgsql;"
	fnV2 := "CREATE OR REPLACE FUNCTION set_updated_at()\nRETURNS TRIGGER AS $$\nBEGIN\n  NEW.updated_at = clock_timestamp();\n  RETURN NEW;\nEND;\n$$ LANGUAGE plpgsql;"

	withV1 := Apply(&Schema{}, fnV1)
	if len(withV1.Functions) != 1 {
		t.Fatalf("after v1: Functions = %d, want 1", len(withV1.Functions))
	}

	withV2 := Apply(withV1, fnV2)
	if len(withV2.Functions) != 1 {
		t.Errorf("Functions: got %d, want 1 (replace must not duplicate)", len(withV2.Functions))
	}
	if withV2.Functions[0].Name != "set_updated_at" {
		t.Errorf("Function name: got %q, want %q", withV2.Functions[0].Name, "set_updated_at")
	}
	// Body must be updated to V2.
	if withV2.Functions[0].Body == withV1.Functions[0].Body {
		t.Error("function body was not updated after CREATE OR REPLACE")
	}
}

func TestApply_ReplaceTrigger(t *testing.T) {
	trigV1 := `DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_users_updated_at') THEN
    CREATE TRIGGER trg_users_updated_at
      BEFORE UPDATE ON users
      FOR EACH ROW EXECUTE FUNCTION set_updated_at();
  END IF;
END $$;`
	trigV2 := `DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_users_updated_at') THEN
    CREATE TRIGGER trg_users_updated_at
      BEFORE INSERT OR UPDATE ON users
      FOR EACH ROW EXECUTE FUNCTION set_updated_at();
  END IF;
END $$;`

	withV1 := Apply(&Schema{}, trigV1)
	if len(withV1.Triggers) != 1 {
		t.Fatalf("after v1: Triggers = %d, want 1", len(withV1.Triggers))
	}

	withV2 := Apply(withV1, trigV2)
	if len(withV2.Triggers) != 1 {
		t.Errorf("Triggers: got %d, want 1 (replace must not duplicate)", len(withV2.Triggers))
	}
	if withV2.Triggers[0].Name != "trg_users_updated_at" {
		t.Errorf("Trigger name: got %q, want %q", withV2.Triggers[0].Name, "trg_users_updated_at")
	}
	// Body must reflect V2.
	if withV2.Triggers[0].Body == withV1.Triggers[0].Body {
		t.Error("trigger body was not updated after re-applying DO block")
	}
}

// ---- Multi-operation sequences ------------------------------------------------

func TestApply_MultipleOperations_SingleSQL(t *testing.T) {
	sql := "CREATE TABLE IF NOT EXISTS products (\n    id UUID PRIMARY KEY DEFAULT gen_random_uuid()\n);\nALTER TABLE products ADD COLUMN name TEXT NOT NULL;\nCREATE INDEX idx_products_name ON products (name);\n"
	result := Apply(&Schema{}, sql)

	if len(result.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(result.Tables))
	}
	if result.Tables[0].Name != "products" {
		t.Errorf("Table name: got %q, want %q", result.Tables[0].Name, "products")
	}
	cols := colsByName(result.Tables[0].Columns)
	if !cols["id"] {
		t.Error("column 'id' missing")
	}
	if !cols["name"] {
		t.Error("column 'name' not added by ALTER TABLE")
	}
	if len(result.Indexes) != 1 {
		t.Fatalf("Indexes: got %d, want 1", len(result.Indexes))
	}
	if result.Indexes[0].Name != "idx_products_name" {
		t.Errorf("Index name: got %q, want %q", result.Indexes[0].Name, "idx_products_name")
	}
}

// ---- DROP TABLE with CASCADE must not affect other tables ---------------------

func TestApply_DropTable_WithCascade(t *testing.T) {
	base := Parse(
		minimalTableSQL("users") +
			minimalTableSQL("posts") +
			minimalTableSQL("comments"),
	)
	if len(base.Tables) != 3 {
		t.Fatalf("precondition: Tables = %d, want 3", len(base.Tables))
	}

	result := Apply(base, "DROP TABLE IF EXISTS users CASCADE;")
	tables := tablesByName(result.Tables)

	if _, ok := tables["users"]; ok {
		t.Error("table 'users' should have been dropped")
	}
	if _, ok := tables["posts"]; !ok {
		t.Error("table 'posts' should still be present after CASCADE drop of users")
	}
	if _, ok := tables["comments"]; !ok {
		t.Error("table 'comments' should still be present after CASCADE drop of users")
	}
	if len(result.Tables) != 2 {
		t.Errorf("Tables: got %d, want 2", len(result.Tables))
	}
}
