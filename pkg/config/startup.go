package config

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"
)

// ConnectWithRetry calls fn up to maxAttempts times, backing off with exponential
// delay and ±25% jitter between attempts. It returns nil on first success.
// If ctx is cancelled during a backoff, it returns immediately with a wrapped error.
// name is used only for log messages.
func ConnectWithRetry(ctx context.Context, name string, logger *slog.Logger, maxAttempts int, fn func() error) error {
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if attempt >= maxAttempts {
			break
		}

		backoff := time.Duration(1<<attempt) * time.Second
		// ±25% jitter to prevent thundering herd on service restart.
		jitter := time.Duration(rand.Int64N(int64(backoff) / 2))
		if rand.IntN(2) == 0 {
			backoff += jitter
		} else {
			backoff -= jitter
		}

		logger.WarnContext(ctx, "service connection failed, retrying",
			"service", name,
			"attempt", attempt,
			"max_attempts", maxAttempts,
			"backoff", backoff,
			"error", err,
		)

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return fmt.Errorf("%s: %w (context canceled during retry)", name, err)
		}
	}
	return fmt.Errorf("%s: failed after %d attempts: %w", name, maxAttempts, err)
}
