package retrybudget_test

import (
	"sync"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/web/retrybudget"
)

func TestBudget_FullBudgetAllowsRetries(t *testing.T) {
	b := retrybudget.New("test", 5)

	// Should be able to take 5 tokens (the burst capacity).
	for i := 0; i < 5; i++ {
		if !b.TryTake() {
			t.Fatalf("TryTake returned false at iteration %d, want true", i)
		}
	}
}

func TestBudget_ExhaustedBudgetReturnsFalse(t *testing.T) {
	b := retrybudget.New("test", 2)

	// Drain the budget.
	b.TryTake()
	b.TryTake()

	// Budget should be exhausted.
	if b.TryTake() {
		t.Fatal("TryTake returned true on exhausted budget, want false")
	}
}

func TestBudget_TokenReplenishmentOverTime(t *testing.T) {
	b := retrybudget.New("test", 100) // 100 tokens/sec

	// Drain the budget.
	for i := 0; i < 100; i++ {
		b.TryTake()
	}
	if b.TryTake() {
		t.Fatal("budget should be exhausted")
	}

	// Wait for tokens to replenish (at 100/s, wait 50ms for ~5 tokens).
	time.Sleep(50 * time.Millisecond)

	if !b.TryTake() {
		t.Fatal("TryTake returned false after replenishment, want true")
	}
}

func TestBudget_ConcurrentSafety(t *testing.T) {
	b := retrybudget.New("test", 1000)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.TryTake()
			b.Remaining()
		}()
	}
	wg.Wait()
}

func TestBudget_Remaining(t *testing.T) {
	b := retrybudget.New("test", 10)
	remaining := b.Remaining()
	if remaining < 0 {
		t.Fatalf("Remaining = %d, want >= 0", remaining)
	}
}
