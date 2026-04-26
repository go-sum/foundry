package ddl

import (
	"strings"
	"testing"
)

// TestColumn_Nullable verifies that Nullable is set correctly based on NOT NULL.
func TestColumn_Nullable(t *testing.T) {
	sql := `CREATE TABLE IF NOT EXISTS things (
    id       UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    required TEXT  NOT NULL,
    optional TEXT
);`
	s := Parse(sql)
	if len(s.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(s.Tables))
	}
	cols := s.Tables[0].Columns
	colMap := make(map[string]Column, len(cols))
	for _, c := range cols {
		colMap[c.Name] = c
	}

	tests := []struct {
		name     string
		nullable bool
	}{
		{"id", false},       // PRIMARY KEY forces non-nullable
		{"required", false}, // explicit NOT NULL
		{"optional", true},  // no NOT NULL constraint
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, ok := colMap[tt.name]
			if !ok {
				t.Fatalf("column %q not found", tt.name)
			}
			if c.Nullable != tt.nullable {
				t.Errorf("column %q Nullable = %v, want %v", tt.name, c.Nullable, tt.nullable)
			}
		})
	}
}

// TestColumn_IsPK verifies PRIMARY KEY detection.
func TestColumn_IsPK(t *testing.T) {
	sql := `CREATE TABLE IF NOT EXISTS things (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL
);`
	s := Parse(sql)
	if len(s.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(s.Tables))
	}
	cols := s.Tables[0].Columns
	colMap := make(map[string]Column, len(cols))
	for _, c := range cols {
		colMap[c.Name] = c
	}

	if !colMap["id"].IsPK {
		t.Error("column id: IsPK = false, want true")
	}
	if colMap["name"].IsPK {
		t.Error("column name: IsPK = true, want false")
	}
}

// TestColumn_IsUnique verifies UNIQUE detection.
func TestColumn_IsUnique(t *testing.T) {
	sql := `CREATE TABLE IF NOT EXISTS things (
    id    UUID  PRIMARY KEY,
    email CITEXT NOT NULL UNIQUE,
    name  TEXT   NOT NULL
);`
	s := Parse(sql)
	if len(s.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(s.Tables))
	}
	cols := s.Tables[0].Columns
	colMap := make(map[string]Column, len(cols))
	for _, c := range cols {
		colMap[c.Name] = c
	}

	if !colMap["email"].IsUnique {
		t.Error("column email: IsUnique = false, want true")
	}
	if colMap["name"].IsUnique {
		t.Error("column name: IsUnique = true, want false")
	}
}

// TestColumn_FKRef verifies that REFERENCES extracts the referenced table.
func TestColumn_FKRef(t *testing.T) {
	sql := `CREATE TABLE IF NOT EXISTS orders (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total   INTEGER NOT NULL DEFAULT 0
);`
	s := Parse(sql)
	if len(s.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(s.Tables))
	}
	cols := s.Tables[0].Columns
	colMap := make(map[string]Column, len(cols))
	for _, c := range cols {
		colMap[c.Name] = c
	}

	if colMap["user_id"].FKRef != "users" {
		t.Errorf("user_id.FKRef = %q, want %q", colMap["user_id"].FKRef, "users")
	}
	if colMap["id"].FKRef != "" {
		t.Errorf("id.FKRef = %q, want empty", colMap["id"].FKRef)
	}
	if colMap["total"].FKRef != "" {
		t.Errorf("total.FKRef = %q, want empty", colMap["total"].FKRef)
	}
}

// TestColumn_TypeNormalized verifies that the Type field is lowercased and has parens stripped.
func TestColumn_TypeNormalized(t *testing.T) {
	sql := `CREATE TABLE IF NOT EXISTS things (
    name    VARCHAR(255) NOT NULL,
    count   INTEGER      NOT NULL DEFAULT 0,
    payload JSONB        NOT NULL
);`
	s := Parse(sql)
	if len(s.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(s.Tables))
	}
	cols := s.Tables[0].Columns
	colMap := make(map[string]Column, len(cols))
	for _, c := range cols {
		colMap[c.Name] = c
	}

	tests := []struct {
		name string
		typ  string
	}{
		{"name", "varchar"},   // parens stripped, lowercased
		{"count", "integer"},  // lowercased
		{"payload", "jsonb"},  // lowercased
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, ok := colMap[tt.name]
			if !ok {
				t.Fatalf("column %q not found", tt.name)
			}
			if c.Type != tt.typ {
				t.Errorf("column %q Type = %q, want %q", tt.name, c.Type, tt.typ)
			}
			// Must not contain parens
			if strings.Contains(c.Type, "(") {
				t.Errorf("column %q Type contains parens: %q", tt.name, c.Type)
			}
		})
	}
}

// TestIndex_IsUnique verifies UNIQUE INDEX detection.
func TestIndex_IsUnique(t *testing.T) {
	sql := `CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_name ON users (name);`
	s := Parse(sql)
	if len(s.Indexes) != 2 {
		t.Fatalf("Indexes: got %d, want 2", len(s.Indexes))
	}
	idxMap := make(map[string]Index, len(s.Indexes))
	for _, idx := range s.Indexes {
		idxMap[idx.Name] = idx
	}

	if !idxMap["idx_users_email"].IsUnique {
		t.Error("idx_users_email: IsUnique = false, want true")
	}
	if idxMap["idx_users_name"].IsUnique {
		t.Error("idx_users_name: IsUnique = true, want false")
	}
}

// TestIndex_Columns verifies that index column names are extracted correctly.
func TestIndex_Columns(t *testing.T) {
	sql := `CREATE INDEX IF NOT EXISTS idx_orders_user_total ON orders (user_id, total);`
	s := Parse(sql)
	if len(s.Indexes) != 1 {
		t.Fatalf("Indexes: got %d, want 1", len(s.Indexes))
	}
	idx := s.Indexes[0]
	if len(idx.Columns) != 2 {
		t.Fatalf("idx.Columns: got %d, want 2 — %v", len(idx.Columns), idx.Columns)
	}
	if idx.Columns[0] != "user_id" {
		t.Errorf("Columns[0] = %q, want %q", idx.Columns[0], "user_id")
	}
	if idx.Columns[1] != "total" {
		t.Errorf("Columns[1] = %q, want %q", idx.Columns[1], "total")
	}
}

// TestIndex_Where verifies that partial index WHERE clause is extracted.
func TestIndex_Where(t *testing.T) {
	sql := `CREATE INDEX IF NOT EXISTS idx_pending ON jobs (queue, run_at)
    WHERE status = 'pending';`
	s := Parse(sql)
	if len(s.Indexes) != 1 {
		t.Fatalf("Indexes: got %d, want 1", len(s.Indexes))
	}
	if s.Indexes[0].Where == "" {
		t.Error("Index.Where is empty, want non-empty for partial index")
	}
}

// TestParse_MultiLineIndex verifies that a CREATE INDEX split across lines is parsed correctly.
func TestParse_MultiLineIndex(t *testing.T) {
	sql := `CREATE INDEX IF NOT EXISTS idx_creds_user_created
    ON webauthn_credentials (user_id, created_at DESC);`
	s := Parse(sql)
	if len(s.Indexes) != 1 {
		t.Fatalf("Indexes: got %d, want 1", len(s.Indexes))
	}
	idx := s.Indexes[0]
	if idx.Name != "idx_creds_user_created" {
		t.Errorf("Index.Name = %q, want %q", idx.Name, "idx_creds_user_created")
	}
	if idx.Table != "webauthn_credentials" {
		t.Errorf("Index.Table = %q, want %q", idx.Table, "webauthn_credentials")
	}
}

// TestParse_FunctionBody verifies that the full function body is preserved in Function.Body.
func TestParse_FunctionBody(t *testing.T) {
	sql := `CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;`
	s := Parse(sql)
	if len(s.Functions) != 1 {
		t.Fatalf("Functions: got %d, want 1", len(s.Functions))
	}
	fn := s.Functions[0]
	if fn.Name != "update_updated_at" {
		t.Errorf("Function.Name = %q, want %q", fn.Name, "update_updated_at")
	}
	if !strings.Contains(fn.Body, "NEW.updated_at = NOW()") {
		t.Errorf("Function.Body missing body content: %q", fn.Body)
	}
	if !strings.Contains(fn.Body, "$$") {
		t.Errorf("Function.Body missing dollar-quote markers: %q", fn.Body)
	}
}

// TestParse_TriggerTable verifies that the trigger's Table field is populated correctly.
func TestParse_TriggerTable(t *testing.T) {
	sql := `DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'orders_updated_at'
    ) THEN
        CREATE TRIGGER orders_updated_at
            BEFORE UPDATE ON orders
            FOR EACH ROW
            EXECUTE FUNCTION update_updated_at();
    END IF;
END $$;`
	s := Parse(sql)
	if len(s.Triggers) != 1 {
		t.Fatalf("Triggers: got %d, want 1", len(s.Triggers))
	}
	trig := s.Triggers[0]
	if trig.Name != "orders_updated_at" {
		t.Errorf("Trigger.Name = %q, want %q", trig.Name, "orders_updated_at")
	}
	if trig.Table != "orders" {
		t.Errorf("Trigger.Table = %q, want %q", trig.Table, "orders")
	}
}

// TestParse_Combined verifies that a SQL block containing multiple construct types
// is fully parsed.
func TestParse_Combined(t *testing.T) {
	sql := `CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS users (
    id    UUID   PRIMARY KEY DEFAULT gen_random_uuid(),
    email CITEXT NOT NULL UNIQUE
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'users_updated_at'
    ) THEN
        CREATE TRIGGER users_updated_at
            BEFORE UPDATE ON users
            FOR EACH ROW
            EXECUTE FUNCTION update_updated_at();
    END IF;
END $$;`

	s := Parse(sql)

	if len(s.Extensions) != 1 || s.Extensions[0] != "citext" {
		t.Errorf("Extensions: got %v, want [citext]", s.Extensions)
	}
	if len(s.Tables) != 1 || s.Tables[0].Name != "users" {
		t.Errorf("Tables: got %v, want [users]", s.Tables)
	}
	if len(s.Indexes) != 1 || s.Indexes[0].Name != "idx_users_email" {
		t.Errorf("Indexes: got %v, want [idx_users_email]", s.Indexes)
	}
	if len(s.Functions) != 1 || s.Functions[0].Name != "update_updated_at" {
		t.Errorf("Functions: got %v, want [update_updated_at]", s.Functions)
	}
	if len(s.Triggers) != 1 || s.Triggers[0].Name != "users_updated_at" {
		t.Errorf("Triggers: got %v, want [users_updated_at]", s.Triggers)
	}
}

// TestParse_ExtensionUnquoted verifies that an unquoted extension name is extracted
// without a trailing semicolon.
func TestParse_ExtensionUnquoted(t *testing.T) {
	sql := `CREATE EXTENSION IF NOT EXISTS pgcrypto;`
	s := Parse(sql)
	if len(s.Extensions) != 1 {
		t.Fatalf("Extensions: got %d, want 1", len(s.Extensions))
	}
	if s.Extensions[0] != "pgcrypto" {
		t.Errorf("Extensions[0] = %q, want %q", s.Extensions[0], "pgcrypto")
	}
}

// TestParse_MultipleExtensions verifies that multiple CREATE EXTENSION statements
// are all captured.
func TestParse_MultipleExtensions(t *testing.T) {
	sql := `CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pgcrypto;`
	s := Parse(sql)
	if len(s.Extensions) != 2 {
		t.Fatalf("Extensions: got %d, want 2", len(s.Extensions))
	}
	extSet := make(map[string]bool)
	for _, e := range s.Extensions {
		extSet[e] = true
	}
	if !extSet["citext"] {
		t.Errorf("missing extension citext; got %v", s.Extensions)
	}
	if !extSet["pgcrypto"] {
		t.Errorf("missing extension pgcrypto; got %v", s.Extensions)
	}
}

// TestParse_TableSchemaPrefix verifies that schema-prefixed table names (e.g. public.users)
// are stored without the schema prefix.
func TestParse_TableSchemaPrefix(t *testing.T) {
	sql := `CREATE TABLE IF NOT EXISTS public.orders (
    id UUID PRIMARY KEY
);`
	s := Parse(sql)
	if len(s.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(s.Tables))
	}
	if s.Tables[0].Name != "orders" {
		t.Errorf("Table.Name = %q, want %q (schema prefix should be stripped)", s.Tables[0].Name, "orders")
	}
}
