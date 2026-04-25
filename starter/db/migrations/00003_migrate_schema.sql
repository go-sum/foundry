-- +goose Up
CREATE TABLE IF NOT EXISTS contact_submissions (
    id text DEFAULT (gen_random_uuid())::text,
    name text NOT NULL,
    email text NOT NULL,
    message text NOT NULL,
    ip_address text DEFAULT '' NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT contact_submissions_pkey PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_contact_submissions_email_created ON contact_submissions (email, created_at DESC);

-- +goose Down
-- REVIEW: auto-generated Down SQL — verify before committing
DROP INDEX IF EXISTS idx_contact_submissions_email_created;
DROP TABLE IF EXISTS contact_submissions CASCADE;
