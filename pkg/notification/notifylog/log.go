package notifylog

import (
	"context"
	"log/slog"

	"github.com/go-sum/foundry/pkg/notification"
)

// Sender emits a structured slog record for each notification. Use in development
// and as an observability channel in production.
type Sender struct {
	logger *slog.Logger
}

// New returns a log Sender. A nil logger falls back to slog.Default().
func New(logger *slog.Logger) *Sender {
	if logger == nil {
		logger = slog.Default()
	}
	return &Sender{logger: logger}
}

// Send implements notification.Sender.
func (s *Sender) Send(ctx context.Context, n notification.Notification) error {
	s.logger.LogAttrs(ctx, slog.LevelInfo, "notification.send",
		slog.String("id", n.ID),
		slog.Int("severity", int(n.Severity)),
		slog.String("subject", n.Subject),
		slog.String("body", n.Body),
		slog.String("request_id", n.Correlation.RequestID),
		slog.String("trace_id", n.Correlation.TraceID),
		slog.String("op", n.Correlation.Op),
		slog.String("subsystem", n.Correlation.Subsystem),
		slog.String("dedupe_key", n.Correlation.DedupeKey),
		slog.String("env", n.Correlation.Env),
		slog.Time("timestamp", n.Timestamp),
	)
	return nil
}
