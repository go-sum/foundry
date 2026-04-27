package db

import (
	"strings"
	"testing"
	"testing/fstest"
)

// --- NewSchema ---

func TestNewSchema_Name(t *testing.T) {
	p := NewSchema("table_a", "SELECT 1", 10)
	if got := p.Name(); got != "table_a" {
		t.Fatalf("Name() = %q, want %q", got, "table_a")
	}
}

func TestNewSchema_SQL(t *testing.T) {
	p := NewSchema("table_a", "SELECT 1", 10)
	if got := p.SQL(); got != "SELECT 1" {
		t.Fatalf("SQL() = %q, want %q", got, "SELECT 1")
	}
}

func TestNewSchema_Priority(t *testing.T) {
	p := NewSchema("table_a", "SELECT 1", 10)
	if got := p.Priority(); got != 10 {
		t.Fatalf("Priority() = %d, want %d", got, 10)
	}
}

// --- BaseSchema ---

func TestBaseSchema_Name(t *testing.T) {
	if got := BaseSchema.Name(); got != "base" {
		t.Fatalf("BaseSchema.Name() = %q, want %q", got, "base")
	}
}

func TestBaseSchema_Priority(t *testing.T) {
	if got := BaseSchema.Priority(); got != 0 {
		t.Fatalf("BaseSchema.Priority() = %d, want %d", got, 0)
	}
}

func TestBaseSchema_SQL_NonEmpty(t *testing.T) {
	if got := BaseSchema.SQL(); got == "" {
		t.Fatal("BaseSchema.SQL() returned empty string; embed likely failed")
	}
}

func TestBaseSchema_SQL_ContainsCitext(t *testing.T) {
	if got := BaseSchema.SQL(); !strings.Contains(got, "citext") {
		t.Fatalf("BaseSchema.SQL() does not contain %q", "citext")
	}
}

func TestBaseSchema_SQL_ContainsUpdateUpdatedAt(t *testing.T) {
	if got := BaseSchema.SQL(); !strings.Contains(got, "update_updated_at") {
		t.Fatalf("BaseSchema.SQL() does not contain %q", "update_updated_at")
	}
}

// --- Registry.Fingerprint ---

// TestRegistry_Fingerprint_Deterministic verifies that two registries built
// from identical providers in the same priority order produce the same fingerprint.
func TestRegistry_Fingerprint_Deterministic(t *testing.T) {
	build := func() *Registry {
		r := NewRegistry()
		r.Register(NewSchema("table_a", "CREATE TABLE a (id int)", 10))
		r.Register(NewSchema("table_b", "CREATE TABLE b (id int)", 20))
		return r
	}

	fp1 := build().Fingerprint()
	fp2 := build().Fingerprint()

	if fp1 != fp2 {
		t.Fatalf("Fingerprint not deterministic: %q != %q", fp1, fp2)
	}
}

// TestRegistry_Fingerprint_ChangesWithSQL verifies that different SQL content
// produces different fingerprints.
func TestRegistry_Fingerprint_ChangesWithSQL(t *testing.T) {
	r1 := NewRegistry()
	r1.Register(NewSchema("table_a", "CREATE TABLE a (id int)", 10))

	r2 := NewRegistry()
	r2.Register(NewSchema("table_a", "CREATE TABLE a (id bigint)", 10))

	fp1 := r1.Fingerprint()
	fp2 := r2.Fingerprint()

	if fp1 == fp2 {
		t.Fatalf("different SQL content produced the same fingerprint: %q", fp1)
	}
}

// TestRegistry_Fingerprint_EmptyRegistry verifies that an empty registry
// returns the SHA-256 of the empty string.
func TestRegistry_Fingerprint_EmptyRegistry(t *testing.T) {
	const wantHex = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	got := NewRegistry().Fingerprint()
	if got != wantHex {
		t.Fatalf("Fingerprint() = %q, want %q", got, wantHex)
	}
}

// TestRegistry_Fingerprint_OrderIndependence verifies that registering providers
// in different insertion orders but the same priorities yields the same fingerprint
// (because Compose sorts by priority before concatenating).
func TestRegistry_Fingerprint_OrderIndependence(t *testing.T) {
	pa := NewSchema("table_a", "CREATE TABLE a (id int)", 10)
	pb := NewSchema("table_b", "CREATE TABLE b (id int)", 20)

	r1 := NewRegistry()
	r1.Register(pa, pb) // a then b

	r2 := NewRegistry()
	r2.Register(pb, pa) // b then a — different insertion order, same priorities

	fp1 := r1.Fingerprint()
	fp2 := r2.Fingerprint()

	if fp1 != fp2 {
		t.Fatalf("insertion order should not affect fingerprint: %q != %q", fp1, fp2)
	}
}

// --- LoadRegistryFromYAML ---

// minimalFS builds a testing/fstest.MapFS with one SQL file at the given path.
func minimalFS(filePath, sql string) fstest.MapFS {
	return fstest.MapFS{
		filePath: &fstest.MapFile{Data: []byte(sql)},
	}
}

// TestLoadRegistryFromYAML_WithResolver_ResolvesExternal verifies that an
// entry whose name matches a resolver key is resolved via the SchemaResolver,
// and that an entry without a resolver match is loaded from the embedded FS.
func TestLoadRegistryFromYAML_WithResolver_ResolvesExternal(t *testing.T) {
	const externalSQL = "CREATE TABLE test_external (id int)"
	const localSQL = "CREATE TABLE local_thing (id int)"

	yamlData := []byte(`schema:
  - name: test
    priority: 5
  - source: sql/local.sql
    priority: 10
`)

	fsys := minimalFS("sql/local.sql", localSQL)

	resolver := SchemaResolver{"test": externalSQL}

	reg, err := LoadRegistryFromYAML(yamlData, fsys, WithResolver(resolver))
	if err != nil {
		t.Fatalf("LoadRegistryFromYAML returned unexpected error: %v", err)
	}

	providers := reg.Providers()
	if len(providers) != 2 {
		t.Fatalf("len(Providers()) = %d, want 2", len(providers))
	}

	// Priority-sorted: "test" (5) before "local_thing" (10).
	if providers[0].Name() != "test" {
		t.Fatalf("providers[0].Name() = %q, want %q", providers[0].Name(), "test")
	}
	if providers[0].SQL() != externalSQL {
		t.Fatalf("providers[0].SQL() = %q, want %q", providers[0].SQL(), externalSQL)
	}
	if providers[0].Priority() != 5 {
		t.Fatalf("providers[0].Priority() = %d, want 5", providers[0].Priority())
	}

	if providers[1].SQL() != localSQL {
		t.Fatalf("providers[1].SQL() = %q, want %q", providers[1].SQL(), localSQL)
	}
	if providers[1].Priority() != 10 {
		t.Fatalf("providers[1].Priority() = %d, want 10", providers[1].Priority())
	}
}

// TestLoadRegistryFromYAML_WithResolver_MissingEntry_Error verifies that a
// YAML entry with no source and no matching resolver key causes an error.
func TestLoadRegistryFromYAML_WithResolver_MissingEntry_Error(t *testing.T) {
	yamlData := []byte(`schema:
  - name: missing
    priority: 1
`)

	resolver := SchemaResolver{} // "missing" key absent

	_, err := LoadRegistryFromYAML(yamlData, fstest.MapFS{}, WithResolver(resolver))
	if err == nil {
		t.Fatal("expected error for missing resolver entry with no source, got nil")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("error message %q does not mention the missing name %q", err.Error(), "missing")
	}
}

// TestLoadRegistryFromYAML_NoExternals_WorksWithoutResolver verifies that a
// YAML with only source-based entries succeeds without a resolver option.
func TestLoadRegistryFromYAML_NoExternals_WorksWithoutResolver(t *testing.T) {
	const sql = "CREATE TABLE things (id int)"

	yamlData := []byte(`schema:
  - source: sql/things.sql
    priority: 10
`)

	fsys := minimalFS("sql/things.sql", sql)

	reg, err := LoadRegistryFromYAML(yamlData, fsys)
	if err != nil {
		t.Fatalf("LoadRegistryFromYAML returned unexpected error: %v", err)
	}

	providers := reg.Providers()
	if len(providers) != 1 {
		t.Fatalf("len(Providers()) = %d, want 1", len(providers))
	}
	if providers[0].SQL() != sql {
		t.Fatalf("providers[0].SQL() = %q, want %q", providers[0].SQL(), sql)
	}
}

// TestLoadRegistryFromYAML_NoSourceNoResolver_Error verifies that an entry
// with no source path and no resolver match returns an error.
func TestLoadRegistryFromYAML_NoSourceNoResolver_Error(t *testing.T) {
	yamlData := []byte(`schema:
  - name: unnamed
    priority: 1
`)

	// Empty resolver: no key will match "unnamed".
	resolver := SchemaResolver{}

	_, err := LoadRegistryFromYAML(yamlData, fstest.MapFS{}, WithResolver(resolver))
	if err == nil {
		t.Fatal("expected error for entry with no source and no resolver match, got nil")
	}
	if !strings.Contains(err.Error(), "unnamed") {
		t.Fatalf("error message %q does not mention the entry name %q", err.Error(), "unnamed")
	}
}

// TestLoadRegistryFromYAML_ResolverTakesPrecedence verifies that when an entry
// has both a name and a source, the resolver SQL is returned rather than the
// file content when the resolver contains the entry name.
func TestLoadRegistryFromYAML_ResolverTakesPrecedence(t *testing.T) {
	const fileSQL = "CREATE TABLE from_file (id int)"
	const resolverSQL = "CREATE TABLE from_resolver (id int)"

	yamlData := []byte(`schema:
  - name: myschema
    source: sql/myschema.sql
    priority: 10
`)

	fsys := minimalFS("sql/myschema.sql", fileSQL)
	resolver := SchemaResolver{"myschema": resolverSQL}

	reg, err := LoadRegistryFromYAML(yamlData, fsys, WithResolver(resolver))
	if err != nil {
		t.Fatalf("LoadRegistryFromYAML returned unexpected error: %v", err)
	}

	providers := reg.Providers()
	if len(providers) != 1 {
		t.Fatalf("len(Providers()) = %d, want 1", len(providers))
	}
	if providers[0].SQL() != resolverSQL {
		t.Fatalf("resolver should take precedence: got %q, want %q", providers[0].SQL(), resolverSQL)
	}
}

// TestLoadRegistryFromYAML_FallbackToFS verifies that when an entry has a
// source path but the resolver does not contain the entry name, the embed.FS
// content is used.
func TestLoadRegistryFromYAML_FallbackToFS(t *testing.T) {
	const fileSQL = "CREATE TABLE from_file (id int)"

	yamlData := []byte(`schema:
  - name: myschema
    source: sql/myschema.sql
    priority: 10
`)

	fsys := minimalFS("sql/myschema.sql", fileSQL)
	// Resolver present but does not contain "myschema" — should fall through to FS.
	resolver := SchemaResolver{"other": "SELECT 1"}

	reg, err := LoadRegistryFromYAML(yamlData, fsys, WithResolver(resolver))
	if err != nil {
		t.Fatalf("LoadRegistryFromYAML returned unexpected error: %v", err)
	}

	providers := reg.Providers()
	if len(providers) != 1 {
		t.Fatalf("len(Providers()) = %d, want 1", len(providers))
	}
	if providers[0].SQL() != fileSQL {
		t.Fatalf("FS fallback should return file content: got %q, want %q", providers[0].SQL(), fileSQL)
	}
}

// TestLoadRegistryFromYAML_NoResolverNoSource_Error verifies that an entry
// with a name but no source and no resolver at all returns an error.
func TestLoadRegistryFromYAML_NoResolverNoSource_Error(t *testing.T) {
	yamlData := []byte(`schema:
  - name: orphan
    priority: 1
`)

	_, err := LoadRegistryFromYAML(yamlData, fstest.MapFS{})
	if err == nil {
		t.Fatal("expected error for entry with no resolver and no source, got nil")
	}
	if !strings.Contains(err.Error(), "orphan") {
		t.Fatalf("error message %q does not mention entry name %q", err.Error(), "orphan")
	}
}
