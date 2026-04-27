package noop

import (
	"context"

	"github.com/go-sum/foundry/pkg/notification"
)

// Sender discards all notifications silently. Use as a safe default in development.
type Sender struct{}

// New returns a noop Sender.
func New() *Sender { return &Sender{} }

// Send implements notification.Sender.
func (*Sender) Send(_ context.Context, _ notification.Notification) error { return nil }
