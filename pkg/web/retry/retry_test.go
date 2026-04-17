package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-sum/web"
	"github.com/go-sum/web/retry"
)

func TestDo(t *testing.T) {
	t.Run("success on first attempt", func(t *testing.T) {
		calls := 0
		err := retry.Do(context.Background(), retry.Policy{MaxAttempts: 3}, func(_ context.Context) error {
			calls++
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls != 1 {
			t.Fatalf("calls = %d, want 1", calls)
		}
	})

	t.Run("success on second attempt after one transient error", func(t *testing.T) {
		calls := 0
		p := retry.Policy{MaxAttempts: 3, BaseDelay: time.Nanosecond, Cap: time.Nanosecond}
		err := retry.Do(context.Background(), p, func(_ context.Context) error {
			calls++
			if calls == 1 {
				return errors.Join(web.ErrTransient, errors.New("temporary"))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls != 2 {
			t.Fatalf("calls = %d, want 2", calls)
		}
	})

	t.Run("non-transient error does not retry", func(t *testing.T) {
		calls := 0
		sentinel := errors.New("permanent")
		p := retry.Policy{MaxAttempts: 3, BaseDelay: time.Nanosecond, Cap: time.Nanosecond}
		err := retry.Do(context.Background(), p, func(_ context.Context) error {
			calls++
			return sentinel
		})
		if !errors.Is(err, sentinel) {
			t.Fatalf("err = %v, want sentinel", err)
		}
		if calls != 1 {
			t.Fatalf("calls = %d, want 1 (no retry for non-transient)", calls)
		}
	})

	t.Run("context cancelled before first attempt", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		calls := 0
		err := retry.Do(ctx, retry.Policy{MaxAttempts: 3}, func(_ context.Context) error {
			calls++
			return nil
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
		if calls != 0 {
			t.Fatalf("calls = %d, want 0", calls)
		}
	})

	t.Run("context cancelled between retries", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		calls := 0
		p := retry.Policy{MaxAttempts: 3, BaseDelay: 50 * time.Millisecond, Cap: 100 * time.Millisecond}
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()
		transientErr := errors.Join(web.ErrTransient, errors.New("temp"))
		err := retry.Do(ctx, p, func(_ context.Context) error {
			calls++
			return transientErr
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	})

	t.Run("max attempts exhausted with transient error", func(t *testing.T) {
		calls := 0
		p := retry.Policy{MaxAttempts: 3, BaseDelay: time.Nanosecond, Cap: time.Nanosecond}
		transientErr := errors.Join(web.ErrTransient, errors.New("temp"))
		err := retry.Do(context.Background(), p, func(_ context.Context) error {
			calls++
			return transientErr
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, web.ErrTransient) {
			t.Fatalf("err = %v, want ErrTransient", err)
		}
		if calls != 3 {
			t.Fatalf("calls = %d, want 3", calls)
		}
	})

	t.Run("budget exhausted stops retries", func(t *testing.T) {
		calls := 0
		p := retry.Policy{
			MaxAttempts: 5,
			BaseDelay:   time.Nanosecond,
			Cap:         time.Nanosecond,
			Budget:      &exhaustedBudget{},
		}
		transientErr := errors.Join(web.ErrTransient, errors.New("temp"))
		err := retry.Do(context.Background(), p, func(_ context.Context) error {
			calls++
			return transientErr
		})
		if !errors.Is(err, web.ErrTransient) {
			t.Fatalf("err = %v, want ErrTransient", err)
		}
		// Budget is checked before second attempt, so only 1 call.
		if calls != 1 {
			t.Fatalf("calls = %d, want 1 (budget exhausted after first attempt)", calls)
		}
	})
}

// exhaustedBudget always returns false (budget exhausted).
type exhaustedBudget struct{}

func (b *exhaustedBudget) TryTake() bool { return false }
