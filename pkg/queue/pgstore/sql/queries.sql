-- Queue job queries.
-- Query names and return modes follow sqlc conventions (-- name: X :one/:exec/:execrows).
-- Generated Go code lives in db/ via the co-located .sqlc.yaml.

-- name: Enqueue :one
INSERT INTO queue_jobs (queue, priority, payload, status, max_attempts, run_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at, updated_at;

-- name: Dequeue :one
-- Atomically claims the highest-priority pending job from the given queues.
-- Uses FOR UPDATE SKIP LOCKED for concurrent-safe worker claiming.
WITH next AS (
    SELECT id FROM queue_jobs
    WHERE queue_jobs.queue = ANY($1::text[])
      AND status = 'pending'
      AND run_at <= NOW()
    ORDER BY priority ASC, run_at ASC
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
UPDATE queue_jobs
SET status = 'running',
    attempts = attempts + 1,
    updated_at = NOW()
FROM next
WHERE queue_jobs.id = next.id
RETURNING queue_jobs.id, queue_jobs.queue, queue_jobs.priority,
          queue_jobs.payload, queue_jobs.status, queue_jobs.attempts,
          queue_jobs.max_attempts, queue_jobs.last_error, queue_jobs.run_at,
          queue_jobs.created_at, queue_jobs.updated_at;

-- name: Complete :exec
UPDATE queue_jobs
SET status = 'completed', updated_at = NOW()
WHERE id = $1;

-- name: Fail :exec
-- Reschedules a job for retry or marks it dead.
UPDATE queue_jobs
SET last_error  = sqlc.arg(last_error),
    updated_at  = NOW(),
    status      = CASE WHEN attempts >= max_attempts THEN 'dead' ELSE 'pending' END,
    run_at      = CASE WHEN attempts >= max_attempts THEN run_at ELSE NOW() + sqlc.arg(retry_after)::interval END
WHERE id = sqlc.arg(id);

-- name: Reap :execrows
-- Reclaims running jobs stuck beyond the stale threshold.
UPDATE queue_jobs
SET status = 'pending', updated_at = NOW()
WHERE id IN (
    SELECT id FROM queue_jobs
    WHERE queue_jobs.queue = ANY(sqlc.arg(queues)::text[])
      AND status = 'running'
      AND updated_at < NOW() - sqlc.arg(stale_after)::interval
    FOR UPDATE SKIP LOCKED
);
