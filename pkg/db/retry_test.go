package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// deadlockErr returns a *pgconn.PgError with the deadlock SQLSTATE.
func deadlockErr() error {
	return &pgconn.PgError{Code: "40P01"}
}

// serializationErr returns a *pgconn.PgError with the serialization failure SQLSTATE.
func serializationErr() error {
	return &pgconn.PgError{Code: "40001"}
}

func TestWithRetry(t *testing.T) {
	const shortDelay = time.Microsecond

	tests := []struct {
		name        string
		maxAttempts int
		// errs is the sequence of errors fn will return; after exhausting the
		// slice, fn returns nil.
		errs        []error
		wantCalls   int
		wantErrIs   error // if non-nil, returned error must satisfy errors.Is(err, wantErrIs)
		wantNilErr  bool  // if true, returned error must be nil
	}{
		{
			name:        "succeeds on first attempt",
			maxAttempts: 3,
			errs:        nil, // fn always succeeds
			wantCalls:   1,
			wantNilErr:  true,
		},
		{
			name:        "non-retryable error returned immediately",
			maxAttempts: 3,
			errs:        []error{errors.New("non-retryable")},
			wantCalls:   1,
			wantErrIs:   nil, // just check non-nil; plain error has no sentinel
		},
		{
			name:        "deadlock on first attempt succeeds on second",
			maxAttempts: 3,
			errs:        []error{deadlockErr()},
			wantCalls:   2,
			wantNilErr:  true,
		},
		{
			name:        "serialization failure then success",
			maxAttempts: 3,
			errs:        []error{serializationErr()},
			wantCalls:   2,
			wantNilErr:  true,
		},
		{
			name:        "all attempts are deadlocks returns last error",
			maxAttempts: 3,
			errs:        []error{deadlockErr(), deadlockErr(), deadlockErr()},
			wantCalls:   3,
			wantNilErr:  false,
		},
		{
			name:        "maxAttempts 1 never retries even on deadlock",
			maxAttempts: 1,
			errs:        []error{deadlockErr()},
			wantCalls:   1,
			wantNilErr:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			errIdx := 0

			fn := func() error {
				calls++
				if errIdx < len(tc.errs) {
					err := tc.errs[errIdx]
					errIdx++
					return err
				}
				return nil
			}

			err := WithRetry(context.Background(), tc.maxAttempts, shortDelay, fn)

			if calls != tc.wantCalls {
				t.Fatalf("fn called %d times, want %d", calls, tc.wantCalls)
			}

			if tc.wantNilErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if !tc.wantNilErr && tc.wantErrIs != nil && !errors.Is(err, tc.wantErrIs) {
				t.Fatalf("expected errors.Is(err, %v); got %v", tc.wantErrIs, err)
			}
			if !tc.wantNilErr && tc.wantErrIs == nil && tc.wantCalls == 1 && len(tc.errs) > 0 {
				// Non-retryable: just confirm an error was returned.
				if err == nil {
					t.Fatal("expected non-nil error for non-retryable failure")
				}
			}
		})
	}
}

func TestWithRetry_AllDeadlocksReturnsDeadlockError(t *testing.T) {
	// Specifically verify that the last deadlock error is what gets returned.
	const shortDelay = time.Microsecond
	const maxAttempts = 3

	calls := 0
	fn := func() error {
		calls++
		return deadlockErr()
	}

	err := WithRetry(context.Background(), maxAttempts, shortDelay, fn)
	if err == nil {
		t.Fatal("expected non-nil error after exhausting attempts")
	}
	if !IsDeadlock(err) {
		t.Fatalf("expected deadlock error, got %v", err)
	}
	if calls != maxAttempts {
		t.Fatalf("fn called %d times, want %d", calls, maxAttempts)
	}
}

func TestWithRetry_ContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before calling WithRetry

	calls := 0
	fn := func() error {
		calls++
		// Return a retryable error so without context check it would retry.
		return deadlockErr()
	}

	// With a cancelled context, the very first call to fn will still happen
	// (WithRetry checks ctx only between retries). After the first deadlock,
	// the select on ctx.Done() fires and ctx.Err() is returned.
	err := WithRetry(ctx, 3, time.Microsecond, fn)
	if err == nil {
		t.Fatal("expected non-nil error from cancelled context")
	}
	// The error should be ctx.Err() (context.Canceled), not the deadlock error.
	if !errors.Is(err, context.Canceled) {
		// If maxAttempts==1 there would be no retry check; but here maxAttempts=3
		// so after the first deadlock the select fires immediately.
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestWithRetry_NonRetryableErrorNotRetried(t *testing.T) {
	sentinel := errors.New("business logic error")
	calls := 0

	fn := func() error {
		calls++
		return sentinel
	}

	err := WithRetry(context.Background(), 5, time.Microsecond, fn)
	if calls != 1 {
		t.Fatalf("fn called %d times for non-retryable error, want 1", calls)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}
