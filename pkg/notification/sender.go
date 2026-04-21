package notification

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// Sender delivers a notification to a single channel.
type Sender interface {
	Send(ctx context.Context, n Notification) error
}

// Dispatcher routes a notification to one or more channel senders.
type Dispatcher struct {
	routes map[Channel]Sender
	logger *slog.Logger
}

// NewDispatcher constructs a Dispatcher. A nil or empty channels map is safe and
// produces a noop dispatcher. A nil logger falls back to slog.Default().
func NewDispatcher(channels map[Channel]Sender, logger *slog.Logger) *Dispatcher {
	if logger == nil {
		logger = slog.Default()
	}
	routes := make(map[Channel]Sender, len(channels))
	for k, v := range channels {
		routes[k] = v
	}
	return &Dispatcher{routes: routes, logger: logger}
}

// Send dispatches n to the channels listed in n.Channels. When n.Channels is
// empty, n is sent to all configured channels. Errors from individual channels
// are joined and returned together.
func (d *Dispatcher) Send(ctx context.Context, n Notification) error {
	if n.Timestamp.IsZero() {
		n.Timestamp = time.Now()
	}
	targets := n.Channels
	if len(targets) == 0 {
		targets = make([]Channel, 0, len(d.routes))
		for ch := range d.routes {
			targets = append(targets, ch)
		}
	}
	var errs []error
	for _, ch := range targets {
		s, ok := d.routes[ch]
		if !ok {
			errs = append(errs, fmt.Errorf("%w: %q", ErrChannelUnavailable, ch))
			continue
		}
		if err := s.Send(ctx, n); err != nil {
			errs = append(errs, fmt.Errorf("notification: channel %q: %w", ch, err))
		}
	}
	return errors.Join(errs...)
}
