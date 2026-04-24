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

// WithDispatcherLogger sets the logger used by the Dispatcher.
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
	purgeInterval time.Duration
	purgeTTL      time.Duration
	purgeBatch    int
	logger        *slog.Logger
}

// WithPollInterval sets how often workers poll the store for new jobs.
func WithPollInterval(d time.Duration) ProcessorOption {
	return func(c *processorCfg) { c.pollInterval = d }
}

// WithShutdownWait sets the maximum time Stop waits for in-flight jobs to finish.
func WithShutdownWait(d time.Duration) ProcessorOption {
	return func(c *processorCfg) { c.shutdownWait = d }
}

// WithReapInterval sets how often the reaper reclaims stuck running jobs.
func WithReapInterval(d time.Duration) ProcessorOption {
	return func(c *processorCfg) { c.reapInterval = d }
}

// WithReapThreshold sets the minimum running duration before a job is reaped.
func WithReapThreshold(d time.Duration) ProcessorOption {
	return func(c *processorCfg) { c.reapThreshold = d }
}

// WithLogger sets the logger used by the Processor.
func WithLogger(l *slog.Logger) ProcessorOption {
	return func(c *processorCfg) { c.logger = l }
}

// WithPurgeInterval sets how often the purger deletes terminal jobs.
func WithPurgeInterval(d time.Duration) ProcessorOption {
	return func(c *processorCfg) { c.purgeInterval = d }
}

// WithPurgeTTL sets the minimum age of terminal jobs before they are purged.
func WithPurgeTTL(d time.Duration) ProcessorOption {
	return func(c *processorCfg) { c.purgeTTL = d }
}

// WithPurgeBatch sets the maximum number of terminal jobs purged per cycle.
func WithPurgeBatch(n int) ProcessorOption {
	return func(c *processorCfg) { c.purgeBatch = n }
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

// newQueueDef returns a queueDef populated with default values.
func newQueueDef(name string, handler HandlerFunc) *queueDef {
	return &queueDef{
		name:        name,
		handler:     handler,
		workers:     1,
		maxAttempts: 3,
		timeout:     30 * time.Second,
		backoff:     5 * time.Second,
		priority:    PriorityDefault,
	}
}

// WithWorkers sets the number of concurrent worker goroutines for the queue.
func WithWorkers(n int) QueueOption {
	return func(q *queueDef) { q.workers = n }
}

// WithMaxAttempts sets the maximum number of delivery attempts before a job is marked dead.
func WithMaxAttempts(n int) QueueOption {
	return func(q *queueDef) { q.maxAttempts = n }
}

// WithTimeout sets the per-job execution deadline.
func WithTimeout(d time.Duration) QueueOption {
	return func(q *queueDef) { q.timeout = d }
}

// WithBackoff sets the base duration for exponential retry backoff.
func WithBackoff(d time.Duration) QueueOption {
	return func(q *queueDef) { q.backoff = d }
}

// WithPriority sets the default scheduling priority for jobs on this queue.
func WithPriority(p Priority) QueueOption {
	return func(q *queueDef) { q.priority = p }
}

// DispatchJobOption overrides per-job dispatch settings.
type DispatchJobOption func(*dispatchJobCfg)

type dispatchJobCfg struct {
	priority *Priority
	runAt    time.Time
}

// RunAt schedules the job to be eligible for dequeue at the given time.
func RunAt(t time.Time) DispatchJobOption {
	return func(c *dispatchJobCfg) { c.runAt = t }
}

// OverridePriority overrides the queue's default priority for a single job.
func OverridePriority(p Priority) DispatchJobOption {
	return func(c *dispatchJobCfg) { c.priority = &p }
}
