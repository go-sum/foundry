package queue

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Dispatcher enqueues jobs. When store is nil (sync mode), registered
// handlers execute inline during Dispatch.
type Dispatcher struct {
	store  Store
	queues map[string]*queueDef
	logger *slog.Logger
	mu     sync.RWMutex
	closed bool
}

// NewDispatcher creates a Dispatcher. Pass nil store for sync mode.
func NewDispatcher(store Store, opts ...DispatcherOption) *Dispatcher {
	var cfg dispatcherCfg
	for _, o := range opts {
		o(&cfg)
	}
	return &Dispatcher{
		store:  store,
		queues: make(map[string]*queueDef),
		logger: cmp.Or(cfg.logger, slog.Default()),
	}
}

// Register associates a handler with a named queue. Must be called before
// the first Dispatch. Panics if called twice for the same queue name.
func (d *Dispatcher) Register(name string, handler HandlerFunc, opts ...QueueOption) {
	if _, exists := d.queues[name]; exists {
		panic(fmt.Sprintf("queue: Register called twice for queue %q", name))
	}
	q := newQueueDef(name, handler)
	for _, o := range opts {
		o(q)
	}
	d.queues[name] = q
}

// Dispatch enqueues a job or, in sync mode, executes the handler inline.
func (d *Dispatcher) Dispatch(ctx context.Context, queue string, payload json.RawMessage, opts ...DispatchJobOption) error {
	d.mu.RLock()
	closed := d.closed
	d.mu.RUnlock()
	if closed {
		return ErrClosed
	}

	def, ok := d.queues[queue]
	if !ok {
		if d.store == nil {
			return ErrQueueUnknown
		}
		def = newQueueDef(queue, nil)
	}

	var jcfg dispatchJobCfg
	for _, o := range opts {
		o(&jcfg)
	}

	priority := def.priority
	if jcfg.priority != nil {
		priority = *jcfg.priority
	}

	runAt := jcfg.runAt
	if runAt.IsZero() {
		runAt = time.Now()
	}

	job := &Job{
		Queue:       queue,
		Priority:    priority,
		Payload:     payload,
		Status:      StatusPending,
		MaxAttempts: def.maxAttempts,
		RunAt:       runAt,
	}

	if d.store == nil {
		return d.dispatchSync(ctx, def, job)
	}
	return d.store.Enqueue(ctx, job)
}

func (d *Dispatcher) dispatchSync(ctx context.Context, def *queueDef, job *Job) error {
	jobCtx, cancel := context.WithTimeout(ctx, def.timeout)
	defer cancel()
	return safeExecute(jobCtx, def.handler, *job)
}

// DispatchPayload marshals payload to JSON then calls Dispatch.
func (d *Dispatcher) DispatchPayload(ctx context.Context, queue string, payload any, opts ...DispatchJobOption) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("queue: marshal payload: %w", err)
	}
	return d.Dispatch(ctx, queue, data, opts...)
}

// Async reports whether the dispatcher has a persistent store.
func (d *Dispatcher) Async() bool {
	return d.store != nil
}

// Close marks the dispatcher as closed; further Dispatch calls return ErrClosed.
func (d *Dispatcher) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return ErrClosed
	}
	d.closed = true
	return nil
}
