package retry

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/go-sum/web"
)

// BudgetChecker allows a retry budget to gate retries.
type BudgetChecker interface {
	TryTake() bool
}

// Policy configures retry behaviour.
type Policy struct {
	// MaxAttempts is the maximum number of attempts (including the first).
	// Zero or negative defaults to 3.
	MaxAttempts int

	// BaseDelay is the initial back-off delay. Defaults to 100ms.
	BaseDelay time.Duration

	// Cap is the maximum back-off delay. Defaults to 2s.
	Cap time.Duration

	// Budget limits the total retry rate across all callers for this upstream.
	// When the budget is exhausted, Do returns without retrying.
	// Nil means no budget.
	Budget BudgetChecker
}

// Do runs fn up to p.MaxAttempts times. It retries only when the returned
// error satisfies errors.Is(err, web.ErrTransient). Between attempts it sleeps
// for a random duration within [0, min(Cap, BaseDelay*2^attempt)] (exponential
// backoff with full jitter). ctx.Err() is checked before each attempt; if the
// context is done, Do returns the context error immediately.
func Do(ctx context.Context, p Policy, fn func(ctx context.Context) error) error {
	if p.MaxAttempts <= 0 {
		p.MaxAttempts = 3
	}
	if p.BaseDelay <= 0 {
		p.BaseDelay = 100 * time.Millisecond
	}
	if p.Cap <= 0 {
		p.Cap = 2 * time.Second
	}

	var lastErr error
	for attempt := 0; attempt < p.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return errors.Join(err, lastErr)
			}
			return err
		}
		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}
		if !errors.Is(lastErr, web.ErrTransient) {
			return lastErr
		}
		if attempt == p.MaxAttempts-1 {
			break
		}
		if p.Budget != nil && !p.Budget.TryTake() {
			return errors.Join(web.ErrTransient, lastErr)
		}
		sleep := jitter(p.BaseDelay, p.Cap, attempt)
		select {
		case <-ctx.Done():
			return errors.Join(ctx.Err(), lastErr)
		case <-time.After(sleep):
		}
	}
	return lastErr
}

// jitter returns a random duration in [0, min(cap, base*2^attempt)].
func jitter(base, cap time.Duration, attempt int) time.Duration {
	maxDelay := base * (1 << attempt) // base * 2^attempt
	if maxDelay > cap || maxDelay < 0 { // overflow guard
		maxDelay = cap
	}
	if maxDelay <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(maxDelay)))
}
