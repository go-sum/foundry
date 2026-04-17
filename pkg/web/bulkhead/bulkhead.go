package bulkhead

import (
	"context"
	"errors"
	"log/slog"

	"github.com/go-sum/web"
)

// ErrExhausted is returned when the bulkhead pool is saturated.
var ErrExhausted = errors.New("bulkhead: pool exhausted")

// Bulkhead limits concurrent access to a resource using a semaphore.
type Bulkhead struct {
	name string
	sem  chan struct{}
}

// New creates a Bulkhead with the given capacity.
func New(name string, capacity int) *Bulkhead {
	if capacity <= 0 {
		capacity = 1
	}
	return &Bulkhead{name: name, sem: make(chan struct{}, capacity)}
}

// Acquire acquires a slot. It returns a release function and nil on success.
// If the pool is saturated, it returns (nil, errors.Join(web.ErrTransient, ErrExhausted))
// immediately without blocking. The caller MUST call release when done.
func (b *Bulkhead) Acquire(_ context.Context) (release func(), err error) {
	select {
	case b.sem <- struct{}{}:
		return func() { <-b.sem }, nil
	default:
		slog.Warn("bulkhead.exhausted", slog.String("subsystem", b.name))
		return nil, errors.Join(web.ErrTransient, ErrExhausted)
	}
}
