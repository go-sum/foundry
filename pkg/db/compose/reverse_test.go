package compose

import (
	"strings"
	"testing"
)

func TestGenerateDown_EmptyInput(t *testing.T) {
	got := GenerateDown("")
	want := "-- REVIEW: auto-generated Down SQL — verify before committing\n"
	if got != want {
		t.Fatalf("GenerateDown(\"\") = %q, want %q", got, want)
	}
}

func TestGenerateDown_AlwaysStartsWithReviewComment(t *testing.T) {
	inputs := []string{
		"",
		"CREATE TABLE foo (id SERIAL PRIMARY KEY);",
		"-- just a comment",
		"SELECT 1;",
	}
	for _, input := range inputs {
		got := GenerateDown(input)
		if !strings.HasPrefix(got, "-- REVIEW: auto-generated Down SQL") {
			t.Fatalf("GenerateDown(%q) does not start with review comment; got %q", input, got)
		}
	}
}

func TestGenerateDown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "CREATE TABLE",
			input: "CREATE TABLE foo (id SERIAL PRIMARY KEY);",
			want:  "-- REVIEW: auto-generated Down SQL — verify before committing\nDROP TABLE IF EXISTS foo CASCADE;\n",
		},
		{
			name:  "CREATE TABLE IF NOT EXISTS",
			input: "CREATE TABLE IF NOT EXISTS foo (col TEXT);",
			want:  "-- REVIEW: auto-generated Down SQL — verify before committing\nDROP TABLE IF EXISTS foo CASCADE;\n",
		},
		{
			name:  "CREATE INDEX",
			input: "CREATE INDEX idx_foo ON foo (col);",
			want:  "-- REVIEW: auto-generated Down SQL — verify before committing\nDROP INDEX IF EXISTS idx_foo;\n",
		},
		{
			name:  "CREATE UNIQUE INDEX",
			input: "CREATE UNIQUE INDEX idx_foo ON foo (col);",
			want:  "-- REVIEW: auto-generated Down SQL — verify before committing\nDROP INDEX IF EXISTS idx_foo;\n",
		},
		{
			name:  "ALTER TABLE ADD COLUMN",
			input: "ALTER TABLE foo ADD COLUMN bar TEXT;",
			want:  "-- REVIEW: auto-generated Down SQL — verify before committing\nALTER TABLE foo DROP COLUMN IF EXISTS bar;\n",
		},
		{
			name:  "ALTER TABLE ADD COLUMN IF NOT EXISTS",
			input: "ALTER TABLE foo ADD COLUMN IF NOT EXISTS bar TEXT;",
			want:  "-- REVIEW: auto-generated Down SQL — verify before committing\nALTER TABLE foo DROP COLUMN IF EXISTS bar;\n",
		},
		{
			name:  "CREATE OR REPLACE FUNCTION",
			input: "CREATE OR REPLACE FUNCTION fn() RETURNS void AS $$ BEGIN END; $$ LANGUAGE plpgsql;",
			want:  "-- REVIEW: auto-generated Down SQL — verify before committing\nDROP FUNCTION IF EXISTS fn();\n",
		},
		{
			name:  "CREATE TRIGGER with ON clause",
			input: "CREATE TRIGGER trg AFTER INSERT ON foo FOR EACH ROW EXECUTE FUNCTION fn();",
			want:  "-- REVIEW: auto-generated Down SQL — verify before committing\nDROP TRIGGER IF EXISTS trg ON foo;\n",
		},
		{
			name: "multiple statements reversed order",
			// CREATE TABLE comes before CREATE INDEX, but in Down output,
			// INDEX is dropped first (reverse order).
			input: "CREATE TABLE foo (id SERIAL PRIMARY KEY);\nCREATE INDEX idx_foo ON foo (id);",
			want: "-- REVIEW: auto-generated Down SQL — verify before committing\n" +
				"DROP INDEX IF EXISTS idx_foo;\n" +
				"DROP TABLE IF EXISTS foo CASCADE;\n",
		},
		{
			name:  "unrecognised statement returns only comment",
			input: "SELECT 1;",
			want:  "-- REVIEW: auto-generated Down SQL — verify before committing\n",
		},
		{
			name:  "comment-only input returns only review comment",
			input: "-- just a comment",
			want:  "-- REVIEW: auto-generated Down SQL — verify before committing\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GenerateDown(tc.input)
			if got != tc.want {
				t.Fatalf("GenerateDown(%q)\n got: %q\nwant: %q", tc.input, got, tc.want)
			}
		})
	}
}
