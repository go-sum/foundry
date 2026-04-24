package queue

import (
	"context"
	"time"
)

// Store is the persistence contract. Must be safe for concurrent use.
type Store interface {
	Enqueue(ctx context.Context, job *Job) error
	Dequeue(ctx context.Context, queues []string) (*Job, error)
	Complete(ctx context.Context, id string) error
	Fail(ctx context.Context, id string, errMsg string, retryAfter time.Duration) error
	// Reap reclaims jobs stuck in running state beyond staleThreshold.
	Reap(ctx context.Context, queues []string, staleThreshold time.Duration) (int, error)
	// Purge deletes completed and dead jobs older than olderThan, up to batchSize at a time.
	Purge(ctx context.Context, olderThan time.Duration, batchSize int) (int, error)
	Ping(ctx context.Context) error
	Close() error
}
