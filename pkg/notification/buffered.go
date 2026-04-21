package notification

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type work struct {
	ctx context.Context
	n   Notification
}

// BufferedDispatcher wraps a Dispatcher with a bounded background send queue.
// The caller must call Shutdown to drain the queue and stop the worker.
type BufferedDispatcher struct {
	inner  *Dispatcher
	queue  chan work
	done   chan struct{}
	mu     sync.Mutex
	closed bool
	logger *slog.Logger
}

// NewBufferedDispatcher constructs a BufferedDispatcher with a background worker.
// A nil logger falls back to slog.Default().
func NewBufferedDispatcher(inner *Dispatcher, queueSize int, logger *slog.Logger) *BufferedDispatcher {
	if logger == nil {
		logger = slog.Default()
	}
	b := &BufferedDispatcher{
		inner:  inner,
		queue:  make(chan work, queueSize),
		done:   make(chan struct{}),
		logger: logger,
	}
	go b.run()
	return b
}

// Send enqueues a notification for background delivery. Returns ErrQueueFull
// when the buffer is exhausted and ErrDeliveryFailed after Shutdown is called.
func (b *BufferedDispatcher) Send(ctx context.Context, n Notification) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrDeliveryFailed
	}
	select {
	case b.queue <- work{ctx: ctx, n: n}:
		b.mu.Unlock()
		return nil
	default:
		b.mu.Unlock()
		return ErrQueueFull
	}
}

// Shutdown closes the queue, drains pending notifications, and stops the
// background worker. Blocks until draining completes or ctx is canceled.
func (b *BufferedDispatcher) Shutdown(ctx context.Context) error {
	b.mu.Lock()
	if !b.closed {
		b.closed = true
		close(b.queue)
	}
	b.mu.Unlock()
	select {
	case <-b.done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("notification: shutdown: %w", ctx.Err())
	}
}

func (b *BufferedDispatcher) run() {
	defer close(b.done)
	for w := range b.queue {
		if err := b.inner.Send(w.ctx, w.n); err != nil {
			b.logger.LogAttrs(w.ctx, slog.LevelError, "notification.dispatch",
				slog.String("cause", err.Error()))
		}
	}
}
