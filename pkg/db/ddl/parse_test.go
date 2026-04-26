package ddl

import (
	"testing"
)

const baseSQL = `-- Base schema: extensions and common trigger functions.
-- Register db.BaseSchema (priority 0) before feature schemas.

CREATE EXTENSION IF NOT EXISTS citext;

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
`

const queueSQL = `-- Queue Jobs schema.

CREATE TABLE IF NOT EXISTS queue_jobs (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    queue        VARCHAR(128) NOT NULL,
    priority     INTEGER      NOT NULL DEFAULT 20,
    payload      JSONB        NOT NULL DEFAULT '{}',
    status       VARCHAR(20)  NOT NULL DEFAULT 'pending',
    attempts     INTEGER      NOT NULL DEFAULT 0,
    max_attempts INTEGER      NOT NULL DEFAULT 3,
    last_error   TEXT         NOT NULL DEFAULT '',
    run_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_queue_jobs_dequeue
    ON queue_jobs (queue, priority, run_at)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_queue_jobs_reap
    ON queue_jobs (status, updated_at)
    WHERE status = 'running';

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'queue_jobs_updated_at'
    ) THEN
        CREATE TRIGGER queue_jobs_updated_at
            BEFORE UPDATE ON queue_jobs
            FOR EACH ROW
            EXECUTE FUNCTION update_updated_at();
    END IF;
END $$;
`

const authSQL = `-- Depends on db.BaseSchema (citext extension, update_updated_at function).

CREATE TABLE IF NOT EXISTS users (
    id            UUID         PRIMARY KEY DEFAULT uuidv7(),
    email         CITEXT       NOT NULL UNIQUE,
    display_name  VARCHAR(255) NOT NULL,
    role          VARCHAR(50)  NOT NULL DEFAULT 'user',
    verified      BOOLEAN      NOT NULL DEFAULT false,
    webauthn_id   BYTEA        UNIQUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);

CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id               UUID         PRIMARY KEY DEFAULT uuidv7(),
    user_id          UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id    BYTEA        NOT NULL UNIQUE,
    name             VARCHAR(255) NOT NULL DEFAULT '',
    public_key       BYTEA        NOT NULL,
    public_key_alg   BIGINT       NOT NULL,
    attestation_type VARCHAR(255) NOT NULL DEFAULT '',
    aaguid           BYTEA        NOT NULL,
    sign_count       BIGINT       NOT NULL DEFAULT 0,
    clone_warning    BOOLEAN      NOT NULL DEFAULT false,
    backup_eligible  BOOLEAN      NOT NULL DEFAULT false,
    backup_state     BOOLEAN      NOT NULL DEFAULT false,
    transports       TEXT[]       NOT NULL DEFAULT '{}',
    attachment       VARCHAR(64)  NOT NULL DEFAULT '',
    last_used_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user_created
    ON webauthn_credentials (user_id, created_at DESC);

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'users_updated_at'
    ) THEN
        CREATE TRIGGER users_updated_at
            BEFORE UPDATE ON users
            FOR EACH ROW
            EXECUTE FUNCTION update_updated_at();
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'webauthn_credentials_updated_at'
    ) THEN
        CREATE TRIGGER webauthn_credentials_updated_at
            BEFORE UPDATE ON webauthn_credentials
            FOR EACH ROW
            EXECUTE FUNCTION update_updated_at();
    END IF;
END $$;
`

const contactSQL = `CREATE TABLE IF NOT EXISTS contact_submissions (
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

func TestParse_Base(t *testing.T) {
	s := Parse(baseSQL)
	if len(s.Extensions) != 1 {
		t.Fatalf("Extensions: got %d, want 1", len(s.Extensions))
	}
	if s.Extensions[0] != "citext" {
		t.Errorf("Extensions[0] = %q, want %q", s.Extensions[0], "citext")
	}
	if len(s.Tables) != 0 {
		t.Errorf("Tables: got %d, want 0", len(s.Tables))
	}
	if len(s.Indexes) != 0 {
		t.Errorf("Indexes: got %d, want 0", len(s.Indexes))
	}
	if len(s.Functions) != 1 {
		t.Fatalf("Functions: got %d, want 1", len(s.Functions))
	}
	if s.Functions[0].Name != "update_updated_at" {
		t.Errorf("Functions[0].Name = %q, want %q", s.Functions[0].Name, "update_updated_at")
	}
	if len(s.Triggers) != 0 {
		t.Errorf("Triggers: got %d, want 0", len(s.Triggers))
	}
}

func TestParse_Queue(t *testing.T) {
	s := Parse(queueSQL)

	if len(s.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(s.Tables))
	}
	if s.Tables[0].Name != "queue_jobs" {
		t.Errorf("Tables[0].Name = %q, want %q", s.Tables[0].Name, "queue_jobs")
	}
	if len(s.Tables[0].Columns) != 11 {
		t.Fatalf("queue_jobs columns: got %d, want 11\ncols: %v", len(s.Tables[0].Columns), s.Tables[0].Columns)
	}

	if len(s.Indexes) != 2 {
		t.Fatalf("Indexes: got %d, want 2", len(s.Indexes))
	}

	var dequeue, reap *Index
	for i := range s.Indexes {
		switch s.Indexes[i].Name {
		case "idx_queue_jobs_dequeue":
			dequeue = &s.Indexes[i]
		case "idx_queue_jobs_reap":
			reap = &s.Indexes[i]
		}
	}
	if dequeue == nil {
		t.Fatal("idx_queue_jobs_dequeue not found")
	}
	if dequeue.Where == "" {
		t.Error("idx_queue_jobs_dequeue.Where should be non-empty (partial index)")
	}
	if reap == nil {
		t.Fatal("idx_queue_jobs_reap not found")
	}
	if reap.Where == "" {
		t.Error("idx_queue_jobs_reap.Where should be non-empty (partial index)")
	}

	if len(s.Triggers) != 1 {
		t.Fatalf("Triggers: got %d, want 1", len(s.Triggers))
	}
	if s.Triggers[0].Name != "queue_jobs_updated_at" {
		t.Errorf("Triggers[0].Name = %q, want %q", s.Triggers[0].Name, "queue_jobs_updated_at")
	}
	if s.Triggers[0].Table != "queue_jobs" {
		t.Errorf("Triggers[0].Table = %q, want %q", s.Triggers[0].Table, "queue_jobs")
	}

	if len(s.Functions) != 0 {
		t.Errorf("Functions: got %d, want 0", len(s.Functions))
	}
}

func TestParse_Auth(t *testing.T) {
	s := Parse(authSQL)

	if len(s.Tables) != 2 {
		t.Fatalf("Tables: got %d, want 2", len(s.Tables))
	}
	names := map[string]bool{}
	for _, tbl := range s.Tables {
		names[tbl.Name] = true
	}
	if !names["users"] {
		t.Error("missing table: users")
	}
	if !names["webauthn_credentials"] {
		t.Error("missing table: webauthn_credentials")
	}

	if len(s.Indexes) != 2 {
		t.Fatalf("Indexes: got %d, want 2", len(s.Indexes))
	}

	if len(s.Triggers) != 2 {
		t.Fatalf("Triggers: got %d, want 2", len(s.Triggers))
	}
	trigNames := map[string]bool{}
	for _, trig := range s.Triggers {
		trigNames[trig.Name] = true
	}
	if !trigNames["users_updated_at"] {
		t.Error("missing trigger: users_updated_at")
	}
	if !trigNames["webauthn_credentials_updated_at"] {
		t.Error("missing trigger: webauthn_credentials_updated_at")
	}

	if len(s.Functions) != 0 {
		t.Errorf("Functions: got %d, want 0", len(s.Functions))
	}
}

func TestParse_Contact(t *testing.T) {
	s := Parse(contactSQL)

	if len(s.Tables) != 1 {
		t.Fatalf("Tables: got %d, want 1", len(s.Tables))
	}
	if s.Tables[0].Name != "contact_submissions" {
		t.Errorf("Tables[0].Name = %q, want %q", s.Tables[0].Name, "contact_submissions")
	}
	if len(s.Tables[0].Columns) != 6 {
		t.Fatalf("contact_submissions columns: got %d, want 6", len(s.Tables[0].Columns))
	}

	if len(s.Indexes) != 1 {
		t.Fatalf("Indexes: got %d, want 1", len(s.Indexes))
	}
	if s.Indexes[0].Name != "idx_contact_submissions_email_created" {
		t.Errorf("Indexes[0].Name = %q, want %q", s.Indexes[0].Name, "idx_contact_submissions_email_created")
	}

	if len(s.Functions) != 0 {
		t.Errorf("Functions: got %d, want 0", len(s.Functions))
	}
	if len(s.Triggers) != 0 {
		t.Errorf("Triggers: got %d, want 0", len(s.Triggers))
	}
}

func TestParse_Empty(t *testing.T) {
	s := Parse("")
	if len(s.Extensions) != 0 {
		t.Errorf("Extensions: got %d, want 0", len(s.Extensions))
	}
	if len(s.Tables) != 0 {
		t.Errorf("Tables: got %d, want 0", len(s.Tables))
	}
	if len(s.Indexes) != 0 {
		t.Errorf("Indexes: got %d, want 0", len(s.Indexes))
	}
	if len(s.Functions) != 0 {
		t.Errorf("Functions: got %d, want 0", len(s.Functions))
	}
	if len(s.Triggers) != 0 {
		t.Errorf("Triggers: got %d, want 0", len(s.Triggers))
	}
}

func TestColumn_Raw(t *testing.T) {
	s := Parse(contactSQL)
	if len(s.Tables) == 0 {
		t.Fatal("no tables parsed")
	}
	cols := s.Tables[0].Columns
	for _, col := range cols {
		if len(col.Raw) == 0 {
			t.Errorf("column %q: Raw is empty", col.Name)
		}
		last := col.Raw[len(col.Raw)-1]
		if last == ',' {
			t.Errorf("column %q: Raw has trailing comma: %q", col.Name, col.Raw)
		}
	}
}

func TestColumn_Default(t *testing.T) {
	s := Parse(queueSQL)
	if len(s.Tables) == 0 {
		t.Fatal("no tables parsed")
	}
	cols := s.Tables[0].Columns
	colMap := make(map[string]Column, len(cols))
	for _, c := range cols {
		colMap[c.Name] = c
	}

	tests := []struct {
		col  string
		want string
	}{
		{"id", "gen_random_uuid()"},
		{"priority", "20"},
		{"payload", "'{}'"},
		{"status", "'pending'"},
		{"attempts", "0"},
		{"last_error", "''"},
	}

	for _, tt := range tests {
		t.Run(tt.col, func(t *testing.T) {
			c, ok := colMap[tt.col]
			if !ok {
				t.Fatalf("column %q not found", tt.col)
			}
			if c.Default != tt.want {
				t.Errorf("column %q Default = %q, want %q", tt.col, c.Default, tt.want)
			}
		})
	}
}
