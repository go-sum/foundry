// Package pgstore implements queue.Store using PostgreSQL with pgx/v5.
// It uses FOR UPDATE SKIP LOCKED for concurrent-safe job claiming.
package pgstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-sum/queue"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Compile-time interface check.
var _ queue.Store = (*Store)(nil)

// Store implements queue.Store backed by PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// New creates a Store. The pool is externally managed and not closed by Close().
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const enqueue = `
INSERT INTO queue_jobs (queue, priority, payload, status, max_attempts, run_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at, updated_at`

// Enqueue inserts a new job and sets its ID from the RETURNING clause.
func (s *Store) Enqueue(ctx context.Context, job *queue.Job) error {
	var id uuid.UUID
	var createdAt, updatedAt time.Time
	err := s.pool.QueryRow(ctx, enqueue,
		job.Queue,
		int32(job.Priority),
		job.Payload,
		string(job.Status),
		int32(job.MaxAttempts),
		job.RunAt,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("pgstore: enqueue: %w", err)
	}
	job.ID = id.String()
	job.CreatedAt = createdAt
	job.UpdatedAt = updatedAt
	return nil
}

const dequeue = `
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
          queue_jobs.created_at, queue_jobs.updated_at`

// Dequeue atomically claims the highest-priority pending job from the given
// queues. Returns queue.ErrNotFound when no work is available.
func (s *Store) Dequeue(ctx context.Context, queues []string) (*queue.Job, error) {
	row := s.pool.QueryRow(ctx, dequeue, queues)
	var j queue.Job
	var id uuid.UUID
	var priority, attempts, maxAttempts int32
	err := row.Scan(&id, &j.Queue, &priority, &j.Payload, &j.Status,
		&attempts, &maxAttempts, &j.LastError, &j.RunAt, &j.CreatedAt, &j.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, queue.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("pgstore: dequeue: %w", err)
	}
	j.ID = id.String()
	j.Priority = queue.Priority(priority)
	j.Attempts = int(attempts)
	j.MaxAttempts = int(maxAttempts)
	return &j, nil
}

const complete = `
UPDATE queue_jobs
SET status = 'completed', updated_at = NOW()
WHERE id = $1`

// Complete marks a job as completed.
func (s *Store) Complete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("pgstore: complete: invalid id: %w", err)
	}
	if _, err := s.pool.Exec(ctx, complete, uid); err != nil {
		return fmt.Errorf("pgstore: complete: %w", err)
	}
	return nil
}

const fail = `
UPDATE queue_jobs
SET last_error  = $1,
    updated_at  = NOW(),
    status      = CASE WHEN attempts >= max_attempts THEN 'dead' ELSE 'pending' END,
    run_at      = CASE WHEN attempts >= max_attempts THEN run_at ELSE NOW() + $2::interval END
WHERE id = $3`

// Fail records a failure. If the job has retries remaining it is rescheduled
// after retryAfter; otherwise it is marked dead.
func (s *Store) Fail(ctx context.Context, id string, errMsg string, retryAfter time.Duration) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("pgstore: fail: invalid id: %w", err)
	}
	if _, err := s.pool.Exec(ctx, fail, errMsg, durationToInterval(retryAfter), uid); err != nil {
		return fmt.Errorf("pgstore: fail: %w", err)
	}
	return nil
}

const reap = `
UPDATE queue_jobs
SET status = 'pending', updated_at = NOW()
WHERE id IN (
    SELECT id FROM queue_jobs
    WHERE queue_jobs.queue = ANY($1::text[])
      AND status = 'running'
      AND updated_at < NOW() - $2::interval
    FOR UPDATE SKIP LOCKED
)`

// Reap reclaims jobs stuck in running state beyond the stale threshold.
func (s *Store) Reap(ctx context.Context, queues []string, staleThreshold time.Duration) (int, error) {
	result, err := s.pool.Exec(ctx, reap, queues, durationToInterval(staleThreshold))
	if err != nil {
		return 0, fmt.Errorf("pgstore: reap: %w", err)
	}
	return int(result.RowsAffected()), nil
}

const purge = `
DELETE FROM queue_jobs
WHERE id IN (
    SELECT id FROM queue_jobs
    WHERE status IN ('completed', 'dead')
      AND updated_at < NOW() - $1::interval
    LIMIT $2
    FOR UPDATE SKIP LOCKED
)`

// Purge deletes completed and dead jobs older than olderThan, up to batchSize at a time.
func (s *Store) Purge(ctx context.Context, olderThan time.Duration, batchSize int) (int, error) {
	result, err := s.pool.Exec(ctx, purge, durationToInterval(olderThan), int32(batchSize))
	if err != nil {
		return 0, fmt.Errorf("pgstore: purge: %w", err)
	}
	return int(result.RowsAffected()), nil
}

// Ping verifies database connectivity.
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Close is a no-op because the pool is externally managed.
func (s *Store) Close() error {
	return nil
}

// durationToInterval converts a time.Duration to a pgtype.Interval for use
// in parameterized queries that accept a PostgreSQL interval type.
func durationToInterval(d time.Duration) pgtype.Interval {
	return pgtype.Interval{
		Microseconds: d.Microseconds(),
		Valid:        true,
	}
}
