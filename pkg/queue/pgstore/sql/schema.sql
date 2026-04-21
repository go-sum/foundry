-- Queue Jobs schema
-- Source of truth for the queue_jobs table. Composed into application
-- migrations via db/sql/schemas.yaml and used by local sqlc generation.

-- Trigger function for automatic updated_at. Uses CREATE OR REPLACE so it
-- coexists safely if the application defines the same function.
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

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
