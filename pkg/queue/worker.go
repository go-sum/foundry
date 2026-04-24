package queue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync/atomic"
	"time"
)

func (p *Processor) runWorker(ctx context.Context, def *queueDef, id int, readyCount *atomic.Int32, totalWorkers int32) {
	defer p.wg.Done()

	timer := time.NewTimer(0)
	defer timer.Stop()

	queues := []string{def.name}
	ready := false

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		if !ready {
			n := readyCount.Add(1)
			ready = true
			if n == totalWorkers && !p.readyClosed.Swap(true) {
				close(p.ready)
			}
		}

		job, err := p.store.Dequeue(ctx, queues)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				timer.Reset(p.cfg.pollInterval)
				continue
			}
			if ctx.Err() != nil {
				return
			}
			p.logger.ErrorContext(ctx, "queue: dequeue error",
				slog.String("queue", def.name),
				slog.Int("worker", id),
				slog.String("error", err.Error()))
			timer.Reset(p.cfg.pollInterval)
			continue
		}

		p.executeJob(ctx, def, job)
		timer.Reset(0)
	}
}

func (p *Processor) executeJob(ctx context.Context, def *queueDef, job *Job) {
	jobCtx, cancel := context.WithTimeout(ctx, def.timeout)
	defer cancel()

	err := safeExecute(jobCtx, def.handler, *job)

	storeCtx, storeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer storeCancel()

	// if NO error
	if err == nil {
		if completeErr := p.store.Complete(storeCtx, job.ID); completeErr != nil {
			p.logger.ErrorContext(ctx, "queue: complete failed",
				slog.String("job_id", job.ID),
				slog.String("error", completeErr.Error()))
		}
		return
	}

	retryAfter := computeBackoff(def.backoff, job.Attempts)
	if failErr := p.store.Fail(storeCtx, job.ID, err.Error(), retryAfter); failErr != nil {
		p.logger.ErrorContext(ctx, "queue: fail record failed",
			slog.String("job_id", job.ID),
			slog.String("error", failErr.Error()))
	}
}

// safeExecute invokes handler with panic recovery.
func safeExecute(ctx context.Context, handler HandlerFunc, job Job) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("queue: handler panic: %v", r)
		}
	}()
	return handler(ctx, job)
}

// computeBackoff returns base * 2^(attempt-1) with ±25% jitter, capped at shift=20.
func computeBackoff(base time.Duration, attempts int) time.Duration {
	shift := attempts - 1
	if shift < 0 {
		shift = 0
	}
	if shift > 20 {
		shift = 20
	}
	d := base * (1 << shift)
	jitter := time.Duration(rand.Int64N(int64(d) / 2))
	if rand.IntN(2) == 0 {
		return d + jitter
	}
	return d - jitter
}

func (p *Processor) runPurger(ctx context.Context) {
	defer p.wg.Done()
	ticker := time.NewTicker(p.cfg.purgeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := p.store.Purge(ctx, p.cfg.purgeTTL, p.cfg.purgeBatch)
			if err != nil {
				p.logger.WarnContext(ctx, "queue: purge error", "error", err)
				continue
			}
			if n > 0 {
				p.logger.InfoContext(ctx, "queue: purged terminal jobs", "count", n)
			}
		}
	}
}

func (p *Processor) runReaper(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.cfg.reapInterval)
	defer ticker.Stop()

	queues := make([]string, 0, len(p.queues))
	for name := range p.queues {
		queues = append(queues, name)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := p.store.Reap(ctx, queues, p.cfg.reapThreshold)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				p.logger.ErrorContext(ctx, "queue: reap error",
					slog.String("error", err.Error()))
				continue
			}
			if n > 0 {
				p.logger.InfoContext(ctx, "queue: reaped stale jobs",
					slog.Int("count", n))
			}
		}
	}
}
