package retrybudget

import (
	"errors"
	"log/slog"
	"sync"
	"time"
)

// ErrBudgetExhausted is returned when the retry budget is exhausted.
var ErrBudgetExhausted = errors.New("retrybudget: exhausted")

// Budget is a sliding-window token bucket that limits the retry rate for a
// single upstream. It is safe for concurrent use.
type Budget struct {
	name    string
	mu      sync.Mutex
	tokens  float64
	maxRate float64 // tokens per nanosecond
	burst   float64 // max token accumulation
	lastAt  time.Time
}

// New creates a Budget allowing at most maxRetriesPerSecond retries per second.
// Burst is set to maxRetriesPerSecond (one second of burst capacity).
func New(name string, maxRetriesPerSecond float64) *Budget {
	if maxRetriesPerSecond <= 0 {
		maxRetriesPerSecond = 1
	}
	rate := maxRetriesPerSecond / float64(time.Second)
	return &Budget{
		name:    name,
		maxRate: rate,
		burst:   maxRetriesPerSecond,
		tokens:  maxRetriesPerSecond, // start full
		lastAt:  time.Now(),
	}
}

// TryTake attempts to consume one retry token. Returns true if the budget
// allows the retry; false if exhausted. Logs a warning on exhaustion.
func (b *Budget) TryTake() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastAt)
	b.lastAt = now

	b.tokens += float64(elapsed) * b.maxRate
	if b.tokens > b.burst {
		b.tokens = b.burst
	}

	if b.tokens < 1 {
		slog.Warn("retry_budget.exhausted", slog.String("subsystem", b.name))
		return false
	}
	b.tokens--
	return true
}

// Remaining returns the approximate number of remaining retry tokens as an
// integer. Useful for metrics and observability.
func (b *Budget) Remaining() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastAt)
	tokens := b.tokens + float64(elapsed)*b.maxRate
	if tokens > b.burst {
		tokens = b.burst
	}
	if tokens < 0 {
		return 0
	}
	return int(tokens)
}
