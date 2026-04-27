package memory

import (
	"context"
	"sync"

	"github.com/go-sum/foundry/pkg/notification"
)

// Sender captures notifications for test assertions.
type Sender struct {
	mu   sync.Mutex
	sent []notification.Notification
}

// New returns a memory Sender.
func New() *Sender { return &Sender{} }

// Send implements notification.Sender.
func (s *Sender) Send(_ context.Context, n notification.Notification) error {
	s.mu.Lock()
	s.sent = append(s.sent, n)
	s.mu.Unlock()
	return nil
}

// Sent returns a copy of all captured notifications (thread-safe).
func (s *Sender) Sent() []notification.Notification {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]notification.Notification, len(s.sent))
	copy(out, s.sent)
	return out
}

// Reset clears all captured notifications.
func (s *Sender) Reset() {
	s.mu.Lock()
	s.sent = s.sent[:0]
	s.mu.Unlock()
}
