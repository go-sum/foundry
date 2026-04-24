// Package pgstore implements queue.Store using PostgreSQL with pgx/v5.
// It uses FOR UPDATE SKIP LOCKED for concurrent-safe job claiming.
package pgstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-sum/queue"
	queuedb "github.com/go-sum/queue/pgstore/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Compile-time interface check.
var _ queue.Store = (*Store)(nil)

// Store implements queue.Store backed by PostgreSQL.
type Store struct {
	pool    *pgxpool.Pool
	queries *queuedb.Queries
}

// New creates a Store. The pool is externally managed and not closed by Close().
func New(pool *pgxpool.Pool) *Store {
	return &Store{
		pool:    pool,
		queries: queuedb.New(pool),
	}
}

// Enqueue inserts a new job and sets its ID from the RETURNING clause.
func (s *Store) Enqueue(ctx context.Context, job *queue.Job) error {
	row, err := s.queries.Enqueue(ctx, queuedb.EnqueueParams{
		Queue:       job.Queue,
		Priority:    int32(job.Priority),
		Payload:     job.Payload,
		Status:      string(job.Status),
		MaxAttempts: int32(job.MaxAttempts),
		RunAt:       job.RunAt,
	})
	if err != nil {
		return fmt.Errorf("pgstore: enqueue: %w", err)
	}
	job.ID = row.ID.String()
	job.CreatedAt = row.CreatedAt
	job.UpdatedAt = row.UpdatedAt
	return nil
}

// Dequeue atomically claims the highest-priority pending job from the given
// queues. Returns queue.ErrNotFound when no work is available.
func (s *Store) Dequeue(ctx context.Context, queues []string) (*queue.Job, error) {
	row, err := s.queries.Dequeue(ctx, queues)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, queue.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("pgstore: dequeue: %w", err)
	}
	return toJobModel(row), nil
}

// Complete marks a job as completed.
func (s *Store) Complete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("pgstore: complete: invalid id: %w", err)
	}
	if err := s.queries.Complete(ctx, uid); err != nil {
		return fmt.Errorf("pgstore: complete: %w", err)
	}
	return nil
}

// Fail records a failure. If the job has retries remaining it is rescheduled
// after retryAfter; otherwise it is marked dead.
func (s *Store) Fail(ctx context.Context, id string, errMsg string, retryAfter time.Duration) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("pgstore: fail: invalid id: %w", err)
	}
	if err := s.queries.Fail(ctx, queuedb.FailParams{
		ID:         uid,
		LastError:  errMsg,
		RetryAfter: durationToInterval(retryAfter),
	}); err != nil {
		return fmt.Errorf("pgstore: fail: %w", err)
	}
	return nil
}

// Reap reclaims jobs stuck in running state beyond the stale threshold.
func (s *Store) Reap(ctx context.Context, queues []string, staleThreshold time.Duration) (int, error) {
	n, err := s.queries.Reap(ctx, queuedb.ReapParams{
		Queues:     queues,
		StaleAfter: durationToInterval(staleThreshold),
	})
	if err != nil {
		return 0, fmt.Errorf("pgstore: reap: %w", err)
	}
	return int(n), nil
}

// Purge deletes completed and dead jobs older than olderThan, up to batchSize at a time.
func (s *Store) Purge(ctx context.Context, olderThan time.Duration, batchSize int) (int, error) {
	n, err := s.queries.Purge(ctx, durationToInterval(olderThan), int32(batchSize))
	if err != nil {
		return 0, fmt.Errorf("pgstore: purge: %w", err)
	}
	return int(n), nil
}

// Ping verifies database connectivity.
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Close is a no-op because the pool is externally managed.
func (s *Store) Close() error {
	return nil
}

// toJobModel converts a sqlc-generated db.QueueJob to the domain model.
func toJobModel(r queuedb.QueueJob) *queue.Job {
	return &queue.Job{
		ID:          r.ID.String(),
		Queue:       r.Queue,
		Priority:    queue.Priority(r.Priority),
		Payload:     r.Payload,
		Status:      queue.Status(r.Status),
		Attempts:    int(r.Attempts),
		MaxAttempts: int(r.MaxAttempts),
		LastError:   r.LastError,
		RunAt:       r.RunAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// durationToInterval converts a time.Duration to a pgtype.Interval for use
// in parameterized queries that accept a PostgreSQL interval type.
func durationToInterval(d time.Duration) pgtype.Interval {
	return pgtype.Interval{
		Microseconds: d.Microseconds(),
		Valid:        true,
	}
}
