package db

import (
	"context"
	"math/rand/v2"
	"time"
)

// DefaultRetryAttempts is the default number of attempts for WithRetry.
const DefaultRetryAttempts = 3

// DefaultRetryDelay is the base backoff delay for WithRetry.
const DefaultRetryDelay = 100 * time.Millisecond

// WithRetry calls fn up to maxAttempts times, retrying only on deadlock or
// serialization failure. Between retries it waits baseDelay * 2^attempt with
// ±25% jitter, capped at 5 seconds. Non-retryable errors are returned immediately.
func WithRetry(ctx context.Context, maxAttempts int, baseDelay time.Duration, fn func() error) error {
	var err error
	for attempt := range maxAttempts {
		err = fn()
		if err == nil {
			return nil
		}
		if !IsDeadlock(err) && !IsSerializationFailure(err) {
			return err
		}
		if attempt == maxAttempts-1 {
			break
		}
		delay := min(baseDelay*(1<<attempt), 5*time.Second)
		// ±25% jitter
		jitter := time.Duration(rand.Int64N(int64(delay) / 2))
		if rand.IntN(2) == 0 {
			delay += jitter
		} else {
			delay -= jitter
		}
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}
