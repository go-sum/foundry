-- +goose Up
CREATE TABLE IF NOT EXISTS queue_jobs (
    id uuid DEFAULT gen_random_uuid(),
    queue varchar(128) NOT NULL,
    priority integer DEFAULT 20 NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    status varchar(20) DEFAULT 'pending' NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    max_attempts integer DEFAULT 3 NOT NULL,
    last_error text DEFAULT '' NOT NULL,
    run_at timestamptz DEFAULT now() NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT queue_jobs_pkey PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_queue_jobs_dequeue ON queue_jobs (queue, priority, run_at) WHERE ((status)::text = 'pending'::text);

CREATE INDEX IF NOT EXISTS idx_queue_jobs_reap ON queue_jobs (status, updated_at) WHERE ((status)::text = 'running'::text);

CREATE OR REPLACE TRIGGER queue_jobs_updated_at
    BEFORE UPDATE ON queue_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- +goose Down
-- REVIEW: auto-generated Down SQL — verify before committing
DROP TRIGGER IF EXISTS queue_jobs_updated_at;
DROP INDEX IF EXISTS idx_queue_jobs_reap;
DROP INDEX IF EXISTS idx_queue_jobs_dequeue;
DROP TABLE IF EXISTS queue_jobs CASCADE;
