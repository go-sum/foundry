package ddl

import (
	"strings"
	"testing"
)

// TestDiff_DroppedTable verifies that a table present in baseline but absent
// in desired produces a DROP TABLE in UpSQL and a CREATE TABLE in DownSQL.
func TestDiff_DroppedTable(t *testing.T) {
	baseline := &Schema{
		Tables: []Table{
			{
				Name: "widgets",
				Columns: []Column{
					{Name: "id", Raw: "id UUID PRIMARY KEY DEFAULT gen_random_uuid()"},
					{Name: "name", Raw: "name TEXT NOT NULL"},
				},
			},
		},
	}
	desired := &Schema{}

	result := Diff(baseline, desired)
	if result.Empty {
		t.Fatal("expected non-empty diff for dropped table")
	}
	if !strings.Contains(result.UpSQL, "DROP TABLE IF EXISTS widgets CASCADE") {
		t.Errorf("UpSQL missing DROP TABLE: %s", result.UpSQL)
	}
	if !strings.Contains(result.DownSQL, "CREATE TABLE IF NOT EXISTS widgets") {
		t.Errorf("DownSQL missing CREATE TABLE: %s", result.DownSQL)
	}
	// DownSQL should reconstruct columns.
	if !strings.Contains(result.DownSQL, "id UUID PRIMARY KEY DEFAULT gen_random_uuid()") {
		t.Errorf("DownSQL missing id column: %s", result.DownSQL)
	}
}

// TestDiff_DroppedColumn verifies that a column present in baseline but absent
// in desired produces ALTER TABLE DROP COLUMN in UpSQL.
func TestDiff_DroppedColumn(t *testing.T) {
	baseline := &Schema{
		Tables: []Table{
			{
				Name: "widgets",
				Columns: []Column{
					{Name: "id", Raw: "id UUID PRIMARY KEY DEFAULT gen_random_uuid()"},
					{Name: "name", Raw: "name TEXT NOT NULL"},
					{Name: "description", Raw: "description TEXT"},
				},
			},
		},
	}
	desired := &Schema{
		Tables: []Table{
			{
				Name: "widgets",
				Columns: []Column{
					{Name: "id", Raw: "id UUID PRIMARY KEY DEFAULT gen_random_uuid()"},
					{Name: "name", Raw: "name TEXT NOT NULL"},
					// description is removed
				},
			},
		},
	}

	result := Diff(baseline, desired)
	if result.Empty {
		t.Fatal("expected non-empty diff for dropped column")
	}
	if !strings.Contains(result.UpSQL, "ALTER TABLE widgets DROP COLUMN IF EXISTS description") {
		t.Errorf("UpSQL missing DROP COLUMN: %s", result.UpSQL)
	}
	if !strings.Contains(result.DownSQL, "ALTER TABLE widgets ADD COLUMN description TEXT") {
		t.Errorf("DownSQL missing ADD COLUMN: %s", result.DownSQL)
	}
}

// TestDiff_DroppedExtension verifies that an extension present in baseline but absent
// in desired produces DROP EXTENSION in UpSQL and CREATE EXTENSION in DownSQL.
func TestDiff_DroppedExtension(t *testing.T) {
	baseline := &Schema{Extensions: []string{"citext"}}
	desired := &Schema{}

	result := Diff(baseline, desired)
	if result.Empty {
		t.Fatal("expected non-empty diff for dropped extension")
	}
	if !strings.Contains(result.UpSQL, "DROP EXTENSION IF EXISTS citext") {
		t.Errorf("UpSQL missing DROP EXTENSION: %s", result.UpSQL)
	}
	if !strings.Contains(result.DownSQL, "CREATE EXTENSION IF NOT EXISTS citext") {
		t.Errorf("DownSQL missing CREATE EXTENSION: %s", result.DownSQL)
	}
}

// TestDiff_DroppedIndex verifies that an index present in baseline but absent
// in desired produces DROP INDEX in UpSQL.
func TestDiff_DroppedIndex(t *testing.T) {
	baseline := &Schema{
		Indexes: []Index{
			{
				Name:  "idx_widgets_name",
				Table: "widgets",
				Raw:   "CREATE INDEX IF NOT EXISTS idx_widgets_name ON widgets (name)",
			},
		},
	}
	desired := &Schema{}

	result := Diff(baseline, desired)
	if result.Empty {
		t.Fatal("expected non-empty diff for dropped index")
	}
	if !strings.Contains(result.UpSQL, "DROP INDEX IF EXISTS idx_widgets_name") {
		t.Errorf("UpSQL missing DROP INDEX: %s", result.UpSQL)
	}
	if !strings.Contains(result.DownSQL, "CREATE INDEX IF NOT EXISTS idx_widgets_name") {
		t.Errorf("DownSQL missing CREATE INDEX: %s", result.DownSQL)
	}
}

// TestDiff_DroppedFunction verifies that a function present in baseline but absent
// in desired produces DROP FUNCTION in UpSQL.
func TestDiff_DroppedFunction(t *testing.T) {
	baseline := &Schema{
		Functions: []Function{
			{
				Name: "update_updated_at",
				Body: "CREATE OR REPLACE FUNCTION update_updated_at()\nRETURNS TRIGGER AS $$\nBEGIN\n    NEW.updated_at = NOW();\n    RETURN NEW;\nEND;\n$$ LANGUAGE plpgsql;",
			},
		},
	}
	desired := &Schema{}

	result := Diff(baseline, desired)
	if result.Empty {
		t.Fatal("expected non-empty diff for dropped function")
	}
	if !strings.Contains(result.UpSQL, "DROP FUNCTION IF EXISTS update_updated_at") {
		t.Errorf("UpSQL missing DROP FUNCTION: %s", result.UpSQL)
	}
	if !strings.Contains(result.DownSQL, "CREATE OR REPLACE FUNCTION update_updated_at") {
		t.Errorf("DownSQL missing function body: %s", result.DownSQL)
	}
}

// TestDiff_UpdatedFunction verifies that a function whose body has changed
// produces an updated function definition in UpSQL.
func TestDiff_UpdatedFunction(t *testing.T) {
	baseline := &Schema{
		Functions: []Function{
			{
				Name: "update_updated_at",
				Body: "CREATE OR REPLACE FUNCTION update_updated_at()\nRETURNS TRIGGER AS $$\nBEGIN\n    NEW.updated_at = NOW();\n    RETURN NEW;\nEND;\n$$ LANGUAGE plpgsql;",
			},
		},
	}
	desired := &Schema{
		Functions: []Function{
			{
				Name: "update_updated_at",
				Body: "CREATE OR REPLACE FUNCTION update_updated_at()\nRETURNS TRIGGER AS $$\nBEGIN\n    NEW.updated_at = clock_timestamp();\n    RETURN NEW;\nEND;\n$$ LANGUAGE plpgsql;",
			},
		},
	}

	result := Diff(baseline, desired)
	if result.Empty {
		t.Fatal("expected non-empty diff for updated function body")
	}
	if !strings.Contains(result.UpSQL, "clock_timestamp()") {
		t.Errorf("UpSQL missing updated function body: %s", result.UpSQL)
	}
}

// TestDiff_IdenticalSchemas_BothNil verifies that two nil schemas produce Empty=true.
func TestDiff_IdenticalSchemas_BothNil(t *testing.T) {
	result := Diff(&Schema{}, &Schema{})
	if !result.Empty {
		t.Errorf("Diff of two empty schemas should be Empty=true; UpSQL: %q", result.UpSQL)
	}
}

// TestDiff_TableWithMultipleColumnChanges verifies that adding and removing columns
// in the same table both produce correct SQL.
func TestDiff_TableWithMultipleColumnChanges(t *testing.T) {
	baseline := &Schema{
		Tables: []Table{
			{
				Name: "items",
				Columns: []Column{
					{Name: "id", Raw: "id UUID PRIMARY KEY"},
					{Name: "old_col", Raw: "old_col TEXT NOT NULL"},
				},
			},
		},
	}
	desired := &Schema{
		Tables: []Table{
			{
				Name: "items",
				Columns: []Column{
					{Name: "id", Raw: "id UUID PRIMARY KEY"},
					{Name: "new_col", Raw: "new_col INTEGER NOT NULL DEFAULT 0"},
				},
			},
		},
	}

	result := Diff(baseline, desired)
	if result.Empty {
		t.Fatal("expected non-empty diff")
	}
	// Adding new_col
	if !strings.Contains(result.UpSQL, "ALTER TABLE items ADD COLUMN new_col INTEGER NOT NULL DEFAULT 0") {
		t.Errorf("UpSQL missing ADD COLUMN new_col: %s", result.UpSQL)
	}
	// Dropping old_col
	if !strings.Contains(result.UpSQL, "ALTER TABLE items DROP COLUMN IF EXISTS old_col") {
		t.Errorf("UpSQL missing DROP COLUMN old_col: %s", result.UpSQL)
	}
}
