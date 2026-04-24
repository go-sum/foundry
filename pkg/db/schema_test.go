package db

import (
	"strings"
	"testing"
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

func TestNewSchema_HealthTables(t *testing.T) {
	p := NewSchema("table_a", "SELECT 1", 10)
	type healthTabler interface {
		HealthTables() []string
	}
	ht, ok := p.(healthTabler)
	if !ok {
		t.Fatal("NewSchema provider does not implement HealthTables() []string")
	}
	tables := ht.HealthTables()
	if len(tables) != 1 || tables[0] != "table_a" {
		t.Fatalf("HealthTables() = %v, want [\"table_a\"]", tables)
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

func TestBaseSchema_DoesNotImplementHealthTables(t *testing.T) {
	type healthTabler interface {
		HealthTables() []string
	}
	var p SchemaProvider = BaseSchema
	if _, ok := p.(healthTabler); ok {
		t.Fatal("BaseSchema must NOT implement HealthTables() []string, but it does")
	}
}

// --- Registry.HealthTables ---

func TestRegistry_HealthTables_Empty(t *testing.T) {
	r := NewRegistry()
	tables := r.HealthTables()
	if len(tables) != 0 {
		t.Fatalf("HealthTables() on empty registry = %v, want nil/empty", tables)
	}
}

func TestRegistry_HealthTables_BaseSchemaOnly(t *testing.T) {
	r := NewRegistry()
	r.Register(BaseSchema)
	tables := r.HealthTables()
	if len(tables) != 0 {
		t.Fatalf("HealthTables() with only BaseSchema = %v, want empty (BaseSchema has no HealthTables)", tables)
	}
}

func TestRegistry_HealthTables_SingleNewSchema(t *testing.T) {
	r := NewRegistry()
	r.Register(NewSchema("table_a", "SELECT 1", 10))
	tables := r.HealthTables()
	if len(tables) != 1 || tables[0] != "table_a" {
		t.Fatalf("HealthTables() = %v, want [\"table_a\"]", tables)
	}
}

func TestRegistry_HealthTables_MixedProviders_PriorityOrder(t *testing.T) {
	// BaseSchema (priority 0) + two NewSchema providers at different priorities.
	// Only NewSchema entries contribute table names; order follows priority.
	r := NewRegistry()
	r.Register(
		NewSchema("table_b", "SELECT 2", 20),
		BaseSchema,
		NewSchema("table_a", "SELECT 1", 10),
	)
	tables := r.HealthTables()
	// Expect priority-sorted: table_a (10) then table_b (20)
	if len(tables) != 2 {
		t.Fatalf("HealthTables() len = %d, want 2; got %v", len(tables), tables)
	}
	if tables[0] != "table_a" {
		t.Fatalf("HealthTables()[0] = %q, want %q", tables[0], "table_a")
	}
	if tables[1] != "table_b" {
		t.Fatalf("HealthTables()[1] = %q, want %q", tables[1], "table_b")
	}
}

func TestRegistry_HealthTables_CustomProvider(t *testing.T) {
	// A custom provider that implements HealthTables.
	r := NewRegistry()
	r.Register(customHealthProvider{})
	tables := r.HealthTables()
	if len(tables) != 2 || tables[0] != "custom_a" || tables[1] != "custom_b" {
		t.Fatalf("HealthTables() = %v, want [\"custom_a\", \"custom_b\"]", tables)
	}
}

// customHealthProvider is a test double that implements SchemaProvider and HealthTables.
type customHealthProvider struct{}

func (customHealthProvider) Name() string            { return "custom" }
func (customHealthProvider) SQL() string             { return "SELECT 'custom'" }
func (customHealthProvider) Priority() int           { return 50 }
func (customHealthProvider) HealthTables() []string  { return []string{"custom_a", "custom_b"} }
