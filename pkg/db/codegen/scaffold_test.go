package codegen

import (
	"strings"
	"testing"
)

// contact_submissions schema used across multiple test cases.
const contactSubmissionsSQL = `
CREATE TABLE IF NOT EXISTS contact_submissions (
    id         TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name       TEXT NOT NULL,
    email      TEXT NOT NULL,
    message    TEXT NOT NULL,
    ip_address TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_contact_submissions_email_created
    ON contact_submissions (email, created_at DESC);
`

func TestParseTables(t *testing.T) {
	tests := []struct {
		name       string
		sql        string
		wantCount  int
		wantTables []TableDef
	}{
		{
			name:      "contact_submissions schema",
			sql:       contactSubmissionsSQL,
			wantCount: 1,
			wantTables: []TableDef{
				{
					Name: "contact_submissions",
					Columns: []ColumnDef{
						{Name: "id", HasDefault: true, IsPK: true},
						{Name: "name", HasDefault: false, IsPK: false},
						{Name: "email", HasDefault: false, IsPK: false},
						{Name: "message", HasDefault: false, IsPK: false},
						{Name: "ip_address", HasDefault: true, IsPK: false},
						{Name: "created_at", HasDefault: true, IsPK: false},
					},
				},
			},
		},
		{
			name: "IF NOT EXISTS variant",
			sql: `
CREATE TABLE IF NOT EXISTS contact_submissions (
    id         TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name       TEXT NOT NULL
);`,
			wantCount: 1,
			wantTables: []TableDef{
				{
					Name: "contact_submissions",
					Columns: []ColumnDef{
						{Name: "id", HasDefault: true, IsPK: true},
						{Name: "name", HasDefault: false, IsPK: false},
					},
				},
			},
		},
		{
			name: "two CREATE TABLEs",
			sql: `
CREATE TABLE users (
    id   TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name TEXT NOT NULL
);

CREATE TABLE posts (
    id      TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    title   TEXT NOT NULL,
    user_id TEXT NOT NULL
);`,
			wantCount: 2,
			wantTables: []TableDef{
				{
					Name: "users",
					Columns: []ColumnDef{
						{Name: "id", HasDefault: true, IsPK: true},
						{Name: "name", HasDefault: false, IsPK: false},
					},
				},
				{
					Name: "posts",
					Columns: []ColumnDef{
						{Name: "id", HasDefault: true, IsPK: true},
						{Name: "title", HasDefault: false, IsPK: false},
						{Name: "user_id", HasDefault: false, IsPK: false},
					},
				},
			},
		},
		{
			name:       "no CREATE TABLE",
			sql:        "-- just a comment\nSELECT 1;",
			wantCount:  0,
			wantTables: nil,
		},
		{
			name: "table-level constraint not parsed as column",
			sql: `
CREATE TABLE example (
    id   TEXT NOT NULL,
    name TEXT NOT NULL,
    PRIMARY KEY (id, name)
);`,
			wantCount: 1,
			wantTables: []TableDef{
				{
					Name: "example",
					Columns: []ColumnDef{
						{Name: "id", HasDefault: false, IsPK: false},
						{Name: "name", HasDefault: false, IsPK: false},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTables(tt.sql)
			if len(got) != tt.wantCount {
				t.Fatalf("ParseTables: got %d tables, want %d", len(got), tt.wantCount)
			}
			for i, want := range tt.wantTables {
				if got[i].Name != want.Name {
					t.Errorf("table[%d].Name = %q, want %q", i, got[i].Name, want.Name)
				}
				if len(got[i].Columns) != len(want.Columns) {
					t.Fatalf("table[%d].Columns len = %d, want %d\ngot:  %+v\nwant: %+v",
						i, len(got[i].Columns), len(want.Columns), got[i].Columns, want.Columns)
				}
				for j, wc := range want.Columns {
					gc := got[i].Columns[j]
					if gc.Name != wc.Name {
						t.Errorf("table[%d].Columns[%d].Name = %q, want %q", i, j, gc.Name, wc.Name)
					}
					if gc.HasDefault != wc.HasDefault {
						t.Errorf("table[%d].Columns[%d].HasDefault = %v, want %v (col %q)", i, j, gc.HasDefault, wc.HasDefault, gc.Name)
					}
					if gc.IsPK != wc.IsPK {
						t.Errorf("table[%d].Columns[%d].IsPK = %v, want %v (col %q)", i, j, gc.IsPK, wc.IsPK, gc.Name)
					}
				}
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"contact_submissions", "ContactSubmissions"},
		{"user", "User"},
		{"queue_jobs", "QueueJobs"},
		{"a", "A"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toPascalCase(tt.input)
			if got != tt.want {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSingularize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"contact_submissions", "contact_submission"},
		{"queue_jobs", "queue_job"},
		{"users", "user"},
		{"address", "address"},
		{"status", "status"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := singularize(tt.input)
			if got != tt.want {
				t.Errorf("singularize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerateGoStore_ContactSubmissions(t *testing.T) {
	tables := ParseTables(contactSubmissionsSQL)
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}

	got := GenerateGoStore(tables[0], "store")

	// Verify package declaration.
	if !strings.Contains(got, "package store") {
		t.Error("missing package declaration")
	}

	// Verify imports are present.
	if !strings.Contains(got, `"context"`) {
		t.Error("missing context import")
	}
	if !strings.Contains(got, `"time"`) {
		t.Error("missing time import")
	}
	if !strings.Contains(got, `"github.com/jackc/pgx/v5"`) {
		t.Error("missing pgx/v5 import")
	}
	if !strings.Contains(got, `"github.com/jackc/pgx/v5/pgxpool"`) {
		t.Error("missing pgxpool import")
	}

	// Verify model struct.
	if !strings.Contains(got, "type ContactSubmission struct") {
		t.Error("missing ContactSubmission struct")
	}

	// Verify scan helper.
	if !strings.Contains(got, "func scanContactSubmission(row pgx.Row)") {
		t.Error("missing scanContactSubmission helper")
	}

	// Verify Store struct and constructor.
	if !strings.Contains(got, "type Store struct") {
		t.Error("missing Store struct")
	}
	if !strings.Contains(got, "func NewStore(pool *pgxpool.Pool) *Store") {
		t.Error("missing NewStore constructor")
	}

	// Verify INSERT method (insert cols: name, email, message, ip_address — id and created_at have defaults).
	if !strings.Contains(got, "func (s *Store) InsertContactSubmission(ctx context.Context") {
		t.Error("missing InsertContactSubmission method")
	}
	if !strings.Contains(got, "INSERT INTO contact_submissions (name, email, message, ip_address)") {
		t.Error("INSERT SQL does not contain expected columns")
	}

	// Verify GET method.
	if !strings.Contains(got, "func (s *Store) GetContactSubmission(ctx context.Context") {
		t.Error("missing GetContactSubmission method")
	}

	// Verify LIST method.
	if !strings.Contains(got, "func (s *Store) ListContactSubmissions(ctx context.Context") {
		t.Error("missing ListContactSubmissions method")
	}
	if !strings.Contains(got, "ORDER BY created_at DESC") {
		t.Error("LIST SQL missing ORDER BY created_at")
	}

	// No UPDATE method since there is no updated_at column.
	if strings.Contains(got, "func (s *Store) UpdateContactSubmission") {
		t.Error("unexpected UpdateContactSubmission — table has no updated_at")
	}

	// Verify DELETE method.
	if !strings.Contains(got, "func (s *Store) DeleteContactSubmission(ctx context.Context") {
		t.Error("missing DeleteContactSubmission method")
	}
}

const fullFeatureSQL = `
CREATE TABLE posts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title      TEXT NOT NULL,
    slug       TEXT NOT NULL UNIQUE,
    body       TEXT NOT NULL,
    author_id  UUID NOT NULL REFERENCES users(id),
    status     VARCHAR(20) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_posts_slug ON posts (slug);

CREATE INDEX IF NOT EXISTS idx_posts_author ON posts (author_id);

CREATE INDEX IF NOT EXISTS idx_posts_status ON posts (status)
    WHERE status = 'published';
`

func TestParseTables_WithIndexes(t *testing.T) {
	tables := ParseTables(fullFeatureSQL)
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}

	td := tables[0]
	if td.Name != "posts" {
		t.Fatalf("table name = %q, want %q", td.Name, "posts")
	}

	// Verify column-level unique and FK detection.
	slugCol := td.Columns[2]
	if slugCol.Name != "slug" {
		t.Fatalf("columns[2].Name = %q, want %q", slugCol.Name, "slug")
	}
	if !slugCol.IsUnique {
		t.Errorf("slug.IsUnique = false, want true")
	}

	authorCol := td.Columns[4]
	if authorCol.Name != "author_id" {
		t.Fatalf("columns[4].Name = %q, want %q", authorCol.Name, "author_id")
	}
	if authorCol.FKRef != "users" {
		t.Errorf("author_id.FKRef = %q, want %q", authorCol.FKRef, "users")
	}

	// Verify indexes.
	if len(td.Indexes) != 3 {
		t.Fatalf("expected 3 indexes, got %d: %+v", len(td.Indexes), td.Indexes)
	}

	idx0 := td.Indexes[0]
	if idx0.Name != "idx_posts_slug" || !idx0.IsUnique || idx0.Where != "" {
		t.Errorf("idx[0] = %+v, want unique non-partial idx_posts_slug", idx0)
	}
	if len(idx0.Columns) != 1 || idx0.Columns[0] != "slug" {
		t.Errorf("idx[0].Columns = %v, want [slug]", idx0.Columns)
	}

	idx1 := td.Indexes[1]
	if idx1.Name != "idx_posts_author" || idx1.IsUnique || idx1.Where != "" {
		t.Errorf("idx[1] = %+v, want non-unique non-partial idx_posts_author", idx1)
	}
	if len(idx1.Columns) != 1 || idx1.Columns[0] != "author_id" {
		t.Errorf("idx[1].Columns = %v, want [author_id]", idx1.Columns)
	}

	idx2 := td.Indexes[2]
	if idx2.Name != "idx_posts_status" || idx2.IsUnique {
		t.Errorf("idx[2] = %+v, want non-unique idx_posts_status", idx2)
	}
	if idx2.Where != "status = 'published'" {
		t.Errorf("idx[2].Where = %q, want %q", idx2.Where, "status = 'published'")
	}
}

func TestGenerateGoStore_FullFeature(t *testing.T) {
	tables := ParseTables(fullFeatureSQL)
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}

	got := GenerateGoStore(tables[0], "posts")

	// Verify package declaration.
	if !strings.Contains(got, "package posts") {
		t.Error("missing package declaration")
	}

	// Verify uuid import since table has UUID columns.
	if !strings.Contains(got, `"github.com/google/uuid"`) {
		t.Error("missing uuid import")
	}

	// Verify time import since table has timestamptz columns.
	if !strings.Contains(got, `"time"`) {
		t.Error("missing time import")
	}

	// Verify Post struct.
	if !strings.Contains(got, "type Post struct") {
		t.Error("missing Post struct")
	}

	// Verify scan helper.
	if !strings.Contains(got, "func scanPost(row pgx.Row)") {
		t.Error("missing scanPost helper")
	}

	// Verify INSERT includes mutable fields (excludes id, created_at, updated_at).
	if !strings.Contains(got, "INSERT INTO posts (title, slug, body, author_id, status)") {
		t.Error("INSERT SQL does not contain expected columns")
	}

	// Verify UPDATE method is present (table has updated_at).
	if !strings.Contains(got, "func (s *Store) UpdatePost(ctx context.Context") {
		t.Error("missing UpdatePost method")
	}

	// Verify DELETE method.
	if !strings.Contains(got, "func (s *Store) DeletePost(ctx context.Context") {
		t.Error("missing DeletePost method")
	}
}

func TestGoType(t *testing.T) {
	tests := []struct {
		sqlType  string
		nullable bool
		want     string
	}{
		{"UUID", false, "uuid.UUID"},
		{"uuid", false, "uuid.UUID"},
		{"uuid", true, "*uuid.UUID"},
		{"TEXT", false, "string"},
		{"text", true, "*string"},
		{"citext", false, "string"},
		{"BOOLEAN", false, "bool"},
		{"bool", true, "*bool"},
		{"INTEGER", false, "int32"},
		{"int", true, "*int32"},
		{"BIGINT", false, "int64"},
		{"TIMESTAMPTZ", false, "time.Time"},
		{"timestamptz", true, "*time.Time"},
		{"JSONB", false, "json.RawMessage"},
		{"jsonb", true, "json.RawMessage"}, // slice types never get pointer
		{"BYTEA", false, "[]byte"},
		{"bytea", true, "[]byte"}, // slice types never get pointer
		{"VARCHAR(20)", false, "string"},
		{"varchar(20)", true, "*string"},
		{"unknown_type", false, "string"}, // safe fallback
	}

	for _, tt := range tests {
		t.Run(tt.sqlType+"_nullable="+boolStr(tt.nullable), func(t *testing.T) {
			got := goType(tt.sqlType, tt.nullable)
			if got != tt.want {
				t.Errorf("goType(%q, %v) = %q, want %q", tt.sqlType, tt.nullable, got, tt.want)
			}
		})
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
