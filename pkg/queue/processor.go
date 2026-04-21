package queue

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// Processor runs worker goroutines that poll the Store for jobs.
// Register queues, then call Start.
type Processor struct {
	store  Store
	queues map[string]*queueDef
	cfg    processorCfg
	logger *slog.Logger

	cancel context.CancelFunc
	wg     sync.WaitGroup
	ready  chan struct{}

	mu          sync.RWMutex
	closed      bool
	readyClosed atomic.Bool
}

// NewProcessor creates a Processor bound to the given store.
func NewProcessor(store Store, opts ...ProcessorOption) *Processor {
	var cfg processorCfg
	for _, o := range opts {
		o(&cfg)
	}
	cfg.pollInterval = cmp.Or(cfg.pollInterval, 1*time.Second)
	cfg.shutdownWait = cmp.Or(cfg.shutdownWait, 30*time.Second)
	cfg.reapInterval = cmp.Or(cfg.reapInterval, 30*time.Second)
	cfg.reapThreshold = cmp.Or(cfg.reapThreshold, 5*time.Minute)

	return &Processor{
		store:  store,
		queues: make(map[string]*queueDef),
		cfg:    cfg,
		logger: cmp.Or(cfg.logger, slog.Default()),
		ready:  make(chan struct{}),
	}
}

// Register adds a queue with its handler and options.
func (p *Processor) Register(name string, handler HandlerFunc, opts ...QueueOption) {
	if _, exists := p.queues[name]; exists {
		panic(fmt.Sprintf("queue: Register called twice for queue %q", name))
	}
	q := &queueDef{
		name:        name,
		handler:     handler,
		workers:     1,
		maxAttempts: 3,
		timeout:     30 * time.Second,
		backoff:     5 * time.Second,
		priority:    PriorityDefault,
	}
	for _, o := range opts {
		o(q)
	}
	p.queues[name] = q
}

// Start launches worker goroutines and the reaper. Non-blocking.
func (p *Processor) Start(ctx context.Context) {
	ctx, p.cancel = context.WithCancel(ctx)

	var totalWorkers int32
	for _, def := range p.queues {
		totalWorkers += int32(def.workers)
	}

	var readyCount atomic.Int32

	for _, def := range p.queues {
		d := def
		for i := range d.workers {
			p.wg.Add(1)
			go p.runWorker(ctx, d, i, &readyCount, totalWorkers)
		}
	}

	p.wg.Add(1)
	go p.runReaper(ctx)

	p.logger.Info("queue processor started", "queues", len(p.queues))
}

// Ready returns a channel that closes once all workers have entered their
// poll loop. Use for test synchronization.
func (p *Processor) Ready() <-chan struct{} {
	return p.ready
}

// Stop signals workers to cease and waits up to ShutdownWait for in-flight
// jobs. Returns ErrClosed on double-stop.
func (p *Processor) Stop() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrClosed
	}
	p.closed = true
	p.mu.Unlock()

	if p.cancel != nil {
		p.cancel()
	}

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(p.cfg.shutdownWait):
		return fmt.Errorf("queue: shutdown timed out after %s", p.cfg.shutdownWait)
	}
}
