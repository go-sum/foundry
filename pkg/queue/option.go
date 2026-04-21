package queue

import (
	"log/slog"
	"time"
)

// DispatcherOption configures a Dispatcher.
type DispatcherOption func(*dispatcherCfg)

type dispatcherCfg struct {
	logger *slog.Logger
}

func WithDispatcherLogger(l *slog.Logger) DispatcherOption {
	return func(c *dispatcherCfg) { c.logger = l }
}

// ProcessorOption configures a Processor.
type ProcessorOption func(*processorCfg)

type processorCfg struct {
	pollInterval  time.Duration
	shutdownWait  time.Duration
	reapInterval  time.Duration
	reapThreshold time.Duration
	logger        *slog.Logger
}

func WithPollInterval(d time.Duration) ProcessorOption {
	return func(c *processorCfg) { c.pollInterval = d }
}

func WithShutdownWait(d time.Duration) ProcessorOption {
	return func(c *processorCfg) { c.shutdownWait = d }
}

func WithReapInterval(d time.Duration) ProcessorOption {
	return func(c *processorCfg) { c.reapInterval = d }
}

func WithReapThreshold(d time.Duration) ProcessorOption {
	return func(c *processorCfg) { c.reapThreshold = d }
}

func WithLogger(l *slog.Logger) ProcessorOption {
	return func(c *processorCfg) { c.logger = l }
}

// QueueOption configures a single queue's behavior.
type QueueOption func(*queueDef)

type queueDef struct {
	name        string
	handler     HandlerFunc
	workers     int
	maxAttempts int
	timeout     time.Duration
	backoff     time.Duration
	priority    Priority
}

func WithWorkers(n int) QueueOption {
	return func(q *queueDef) { q.workers = n }
}

func WithMaxAttempts(n int) QueueOption {
	return func(q *queueDef) { q.maxAttempts = n }
}

func WithTimeout(d time.Duration) QueueOption {
	return func(q *queueDef) { q.timeout = d }
}

func WithBackoff(d time.Duration) QueueOption {
	return func(q *queueDef) { q.backoff = d }
}

func WithPriority(p Priority) QueueOption {
	return func(q *queueDef) { q.priority = p }
}

// DispatchJobOption overrides per-job dispatch settings.
type DispatchJobOption func(*dispatchJobCfg)

type dispatchJobCfg struct {
	priority *Priority
	runAt    time.Time
}

func RunAt(t time.Time) DispatchJobOption {
	return func(c *dispatchJobCfg) { c.runAt = t }
}

func OverridePriority(p Priority) DispatchJobOption {
	return func(c *dispatchJobCfg) { c.priority = &p }
}
