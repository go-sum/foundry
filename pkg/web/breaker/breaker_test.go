package breaker_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/breaker"
)

var transientErr = errors.Join(web.ErrTransient, errors.New("upstream error"))

func TestBreaker_ClosedPassesThrough(t *testing.T) {
	b := breaker.New(breaker.Config{Name: "test", FailureThreshold: 3})
	called := false
	err := b.Do(context.Background(), func(_ context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("fn was not called")
	}
}

func TestBreaker_OpensAfterThresholdTransientFailures(t *testing.T) {
	b := breaker.New(breaker.Config{
		Name:             "test",
		FailureThreshold: 3,
		Window:           10 * time.Second,
		Recovery:         30 * time.Second,
	})

	// Drive 3 transient failures to open the breaker.
	for i := 0; i < 3; i++ {
		_ = b.Do(context.Background(), func(_ context.Context) error {
			return transientErr
		})
	}

	// Next call should be rejected.
	err := b.Do(context.Background(), func(_ context.Context) error {
		return nil
	})
	if !errors.Is(err, breaker.ErrBreakerOpen) {
		t.Fatalf("err = %v, want ErrBreakerOpen", err)
	}
	if !errors.Is(err, web.ErrTransient) {
		t.Fatalf("err = %v, want web.ErrTransient", err)
	}
}

func TestBreaker_OpenRejectsImmediately(t *testing.T) {
	b := breaker.New(breaker.Config{
		Name:             "test",
		FailureThreshold: 1,
		Recovery:         1 * time.Hour, // won't recover during test
	})

	// Open the breaker.
	_ = b.Do(context.Background(), func(_ context.Context) error {
		return transientErr
	})

	calls := 0
	err := b.Do(context.Background(), func(_ context.Context) error {
		calls++
		return nil
	})
	if !errors.Is(err, breaker.ErrBreakerOpen) {
		t.Fatalf("err = %v, want ErrBreakerOpen", err)
	}
	if calls != 0 {
		t.Fatalf("calls = %d, want 0 (fn should not be called when open)", calls)
	}
}

func TestBreaker_HalfOpenAfterRecovery(t *testing.T) {
	b := breaker.New(breaker.Config{
		Name:             "test",
		FailureThreshold: 1,
		Recovery:         10 * time.Millisecond,
	})

	// Open the breaker.
	_ = b.Do(context.Background(), func(_ context.Context) error {
		return transientErr
	})

	// Wait for recovery window.
	time.Sleep(20 * time.Millisecond)

	// Probe should be allowed.
	called := false
	err := b.Do(context.Background(), func(_ context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error during probe: %v", err)
	}
	if !called {
		t.Fatal("probe fn was not called")
	}
}

func TestBreaker_SuccessfulProbeCloses(t *testing.T) {
	b := breaker.New(breaker.Config{
		Name:             "test",
		FailureThreshold: 1,
		Recovery:         10 * time.Millisecond,
	})

	// Open then wait for recovery.
	_ = b.Do(context.Background(), func(_ context.Context) error {
		return transientErr
	})
	time.Sleep(20 * time.Millisecond)

	// Successful probe should close the breaker.
	_ = b.Do(context.Background(), func(_ context.Context) error { return nil })

	// Subsequent calls should be allowed.
	calls := 0
	err := b.Do(context.Background(), func(_ context.Context) error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error after breaker closed: %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestBreaker_FailedProbeResetsRecoveryWindow(t *testing.T) {
	b := breaker.New(breaker.Config{
		Name:             "test",
		FailureThreshold: 1,
		Recovery:         20 * time.Millisecond,
	})

	// Open the breaker.
	_ = b.Do(context.Background(), func(_ context.Context) error {
		return transientErr
	})

	// Wait for recovery, then send a failing probe.
	time.Sleep(25 * time.Millisecond)
	_ = b.Do(context.Background(), func(_ context.Context) error {
		return transientErr
	})

	// Breaker should still be open (recovery window reset).
	err := b.Do(context.Background(), func(_ context.Context) error { return nil })
	if !errors.Is(err, breaker.ErrBreakerOpen) {
		t.Fatalf("err = %v, want ErrBreakerOpen after failed probe", err)
	}
}

func TestBreaker_NonTransientErrorsDoNotCount(t *testing.T) {
	b := breaker.New(breaker.Config{
		Name:             "test",
		FailureThreshold: 2,
		Window:           10 * time.Second,
	})

	permanent := errors.New("permanent")
	for i := 0; i < 5; i++ {
		_ = b.Do(context.Background(), func(_ context.Context) error {
			return permanent
		})
	}

	// Breaker should remain closed.
	called := false
	err := b.Do(context.Background(), func(_ context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("fn should have been called (non-transient errors don't open breaker)")
	}
}

func TestBreaker_ConcurrentSafety(t *testing.T) {
	b := breaker.New(breaker.Config{
		Name:             "test",
		FailureThreshold: 100,
		Window:           10 * time.Second,
	})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Do(context.Background(), func(_ context.Context) error {
				return transientErr
			})
		}()
	}
	wg.Wait()
}
