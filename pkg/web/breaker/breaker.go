package breaker

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/go-sum/web"
)

// ErrBreakerOpen is returned when a call is rejected because the breaker is open.
var ErrBreakerOpen = errors.New("breaker: open")

type state int

const (
	stateClosed   state = iota
	stateOpen
	stateHalfOpen
)

// Config configures a Breaker.
type Config struct {
	// Name identifies the upstream in logs.
	Name string

	// FailureThreshold is the number of transient failures within Window
	// required to open the breaker. Defaults to 5.
	FailureThreshold int

	// Window is the sliding failure-counting window. Defaults to 10s.
	Window time.Duration

	// Recovery is the time to wait in Open state before probing. Defaults to 30s.
	Recovery time.Duration

	// Logger is used for state-change events. Defaults to slog.Default().
	Logger *slog.Logger
}

// Breaker wraps calls to an upstream and opens when sustained transient
// failures exceed the configured threshold.
type Breaker struct {
	cfg      Config
	mu       sync.Mutex
	state    state
	failures []time.Time // timestamps of recent transient failures
	openedAt time.Time
	logger   *slog.Logger
}

// New creates a Breaker with the given config.
func New(cfg Config) *Breaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.Window <= 0 {
		cfg.Window = 10 * time.Second
	}
	if cfg.Recovery <= 0 {
		cfg.Recovery = 30 * time.Second
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Breaker{cfg: cfg, logger: logger}
}

// Do executes fn if the breaker is closed or half-open (probe).
// When open, Do returns errors.Join(web.ErrTransient, ErrBreakerOpen) immediately.
// A successful probe closes the breaker; a failed probe resets the recovery window.
func (b *Breaker) Do(ctx context.Context, fn func(context.Context) error) error {
	b.mu.Lock()
	allow, probe := b.checkLocked(time.Now())
	b.mu.Unlock()

	if !allow {
		return errors.Join(web.ErrTransient, ErrBreakerOpen)
	}

	err := fn(ctx)

	b.mu.Lock()
	b.recordLocked(err, probe, time.Now())
	b.mu.Unlock()

	return err
}

func (b *Breaker) checkLocked(now time.Time) (allow bool, probe bool) {
	switch b.state {
	case stateClosed:
		return true, false
	case stateOpen:
		if now.Sub(b.openedAt) >= b.cfg.Recovery {
			b.state = stateHalfOpen
			b.logger.Info("breaker.half_open", slog.String("subsystem", b.cfg.Name))
			return true, true
		}
		return false, false
	case stateHalfOpen:
		return true, true
	}
	return false, false
}

func (b *Breaker) recordLocked(err error, probe bool, now time.Time) {
	if err == nil {
		if probe || b.state == stateHalfOpen {
			b.state = stateClosed
			b.failures = b.failures[:0]
			b.logger.Info("breaker.closed", slog.String("subsystem", b.cfg.Name))
		}
		return
	}
	if !errors.Is(err, web.ErrTransient) {
		return // non-transient failure doesn't count
	}

	if probe {
		// Failed probe: reset recovery window.
		b.openedAt = now
		b.state = stateOpen
		b.logger.Warn("breaker.open",
			slog.String("subsystem", b.cfg.Name),
			slog.String("cause", err.Error()),
		)
		return
	}

	// Prune failures outside the window.
	cutoff := now.Add(-b.cfg.Window)
	valid := b.failures[:0]
	for _, t := range b.failures {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	b.failures = append(valid, now)

	if len(b.failures) >= b.cfg.FailureThreshold {
		b.state = stateOpen
		b.openedAt = now
		b.logger.Warn("breaker.open",
			slog.String("subsystem", b.cfg.Name),
			slog.String("failures", "threshold_exceeded"),
		)
	}
}
