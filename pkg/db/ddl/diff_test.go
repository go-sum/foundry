package ddl

import (
	"strings"
	"testing"
)

func TestDiff_Empty(t *testing.T) {
	s := Parse(contactSQL)
	result := Diff(s, s)
	if !result.Empty {
		t.Errorf("Diff of identical schemas should be Empty=true\nUpSQL: %s", result.UpSQL)
	}
}

func TestDiff_AllNew(t *testing.T) {
	baseline := &Schema{}
	desired := Parse(baseSQL)
	result := Diff(baseline, desired)
	if result.Empty {
		t.Fatal("expected non-empty diff")
	}
	if !strings.Contains(result.UpSQL, "CREATE EXTENSION IF NOT EXISTS citext") {
		t.Errorf("UpSQL missing extension: %s", result.UpSQL)
	}
	if !strings.Contains(result.UpSQL, "CREATE OR REPLACE FUNCTION") {
		t.Errorf("UpSQL missing function: %s", result.UpSQL)
	}
}

func TestDiff_NewTable(t *testing.T) {
	baseline := &Schema{}
	desired := &Schema{
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
	result := Diff(baseline, desired)
	if result.Empty {
		t.Fatal("expected non-empty diff")
	}
	if !strings.Contains(result.UpSQL, "CREATE TABLE IF NOT EXISTS widgets") {
		t.Errorf("UpSQL missing CREATE TABLE: %s", result.UpSQL)
	}
	if !strings.Contains(result.UpSQL, "id UUID PRIMARY KEY DEFAULT gen_random_uuid()") {
		t.Errorf("UpSQL missing id column: %s", result.UpSQL)
	}
	if !strings.Contains(result.DownSQL, "DROP TABLE IF EXISTS widgets CASCADE") {
		t.Errorf("DownSQL missing DROP TABLE: %s", result.DownSQL)
	}
}

func TestDiff_NewColumn(t *testing.T) {
	baseline := &Schema{
		Tables: []Table{
			{
				Name: "widgets",
				Columns: []Column{
					{Name: "id", Raw: "id UUID PRIMARY KEY DEFAULT gen_random_uuid()"},
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
				},
			},
		},
	}
	result := Diff(baseline, desired)
	if result.Empty {
		t.Fatal("expected non-empty diff")
	}
	if !strings.Contains(result.UpSQL, "ALTER TABLE widgets ADD COLUMN name TEXT NOT NULL") {
		t.Errorf("UpSQL missing ADD COLUMN: %s", result.UpSQL)
	}
	if !strings.Contains(result.DownSQL, "ALTER TABLE widgets DROP COLUMN IF EXISTS name") {
		t.Errorf("DownSQL missing DROP COLUMN: %s", result.DownSQL)
	}
}

func TestDiff_NewIndex(t *testing.T) {
	baseline := &Schema{}
	desired := &Schema{
		Indexes: []Index{
			{
				Name:  "idx_widgets_name",
				Table: "widgets",
				Raw:   "CREATE INDEX IF NOT EXISTS idx_widgets_name ON widgets (name)",
			},
		},
	}
	result := Diff(baseline, desired)
	if result.Empty {
		t.Fatal("expected non-empty diff")
	}
	if !strings.Contains(result.UpSQL, "CREATE INDEX IF NOT EXISTS idx_widgets_name") {
		t.Errorf("UpSQL missing CREATE INDEX: %s", result.UpSQL)
	}
	if !strings.Contains(result.DownSQL, "DROP INDEX IF EXISTS idx_widgets_name") {
		t.Errorf("DownSQL missing DROP INDEX: %s", result.DownSQL)
	}
}

func TestDiff_NewExtension(t *testing.T) {
	baseline := &Schema{}
	desired := &Schema{Extensions: []string{"citext"}}
	result := Diff(baseline, desired)
	if result.Empty {
		t.Fatal("expected non-empty diff")
	}
	if !strings.Contains(result.UpSQL, "CREATE EXTENSION IF NOT EXISTS citext") {
		t.Errorf("UpSQL missing extension: %s", result.UpSQL)
	}
	if !strings.Contains(result.DownSQL, "DROP EXTENSION IF EXISTS citext") {
		t.Errorf("DownSQL missing drop extension: %s", result.DownSQL)
	}
}

func TestDiff_NewFunction(t *testing.T) {
	baseline := &Schema{}
	desired := &Schema{
		Functions: []Function{
			{
				Name: "update_updated_at",
				Body: "CREATE OR REPLACE FUNCTION update_updated_at()\nRETURNS TRIGGER AS $$\nBEGIN\n    NEW.updated_at = NOW();\n    RETURN NEW;\nEND;\n$$ LANGUAGE plpgsql;",
			},
		},
	}
	result := Diff(baseline, desired)
	if result.Empty {
		t.Fatal("expected non-empty diff")
	}
	if !strings.Contains(result.UpSQL, "CREATE OR REPLACE FUNCTION update_updated_at") {
		t.Errorf("UpSQL missing function: %s", result.UpSQL)
	}
	if !strings.Contains(result.DownSQL, "DROP FUNCTION IF EXISTS update_updated_at") {
		t.Errorf("DownSQL missing drop function: %s", result.DownSQL)
	}
}

func TestDiff_NewTrigger(t *testing.T) {
	baseline := &Schema{}
	desired := &Schema{
		Triggers: []Trigger{
			{
				Name:  "queue_jobs_updated_at",
				Table: "queue_jobs",
				Body:  "DO $$ BEGIN\n    IF NOT EXISTS (\n        SELECT 1 FROM pg_trigger WHERE tgname = 'queue_jobs_updated_at'\n    ) THEN\n        CREATE TRIGGER queue_jobs_updated_at\n            BEFORE UPDATE ON queue_jobs\n            FOR EACH ROW\n            EXECUTE FUNCTION update_updated_at();\n    END IF;\nEND $$;",
			},
		},
	}
	result := Diff(baseline, desired)
	if result.Empty {
		t.Fatal("expected non-empty diff")
	}
	if !strings.Contains(result.UpSQL, "DO $$") {
		t.Errorf("UpSQL missing DO block: %s", result.UpSQL)
	}
	if !strings.Contains(result.DownSQL, "DROP TRIGGER IF EXISTS queue_jobs_updated_at ON queue_jobs") {
		t.Errorf("DownSQL missing drop trigger: %s", result.DownSQL)
	}
}

func TestDiff_DownReverses(t *testing.T) {
	baseline := &Schema{}
	desired := Parse(queueSQL)
	result := Diff(baseline, desired)

	if result.Empty {
		t.Fatal("expected non-empty diff")
	}

	// Up should create the table.
	if !strings.Contains(result.UpSQL, "CREATE TABLE IF NOT EXISTS queue_jobs") {
		t.Errorf("UpSQL missing CREATE TABLE: %s", result.UpSQL)
	}

	// Up should create both indexes.
	if !strings.Contains(result.UpSQL, "idx_queue_jobs_dequeue") {
		t.Errorf("UpSQL missing dequeue index: %s", result.UpSQL)
	}
	if !strings.Contains(result.UpSQL, "idx_queue_jobs_reap") {
		t.Errorf("UpSQL missing reap index: %s", result.UpSQL)
	}

	// Up should include trigger DO block.
	if !strings.Contains(result.UpSQL, "queue_jobs_updated_at") {
		t.Errorf("UpSQL missing trigger: %s", result.UpSQL)
	}

	// Down should drop table.
	if !strings.Contains(result.DownSQL, "DROP TABLE IF EXISTS queue_jobs CASCADE") {
		t.Errorf("DownSQL missing DROP TABLE: %s", result.DownSQL)
	}

	// Down should drop indexes.
	if !strings.Contains(result.DownSQL, "DROP INDEX IF EXISTS idx_queue_jobs_dequeue") {
		t.Errorf("DownSQL missing drop dequeue index: %s", result.DownSQL)
	}
	if !strings.Contains(result.DownSQL, "DROP INDEX IF EXISTS idx_queue_jobs_reap") {
		t.Errorf("DownSQL missing drop reap index: %s", result.DownSQL)
	}

	// Down should drop trigger.
	if !strings.Contains(result.DownSQL, "DROP TRIGGER IF EXISTS queue_jobs_updated_at ON queue_jobs") {
		t.Errorf("DownSQL missing drop trigger: %s", result.DownSQL)
	}
}
