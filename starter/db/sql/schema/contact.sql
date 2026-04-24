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
