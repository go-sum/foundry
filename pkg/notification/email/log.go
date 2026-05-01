package email

import (
	"context"
	"log/slog"
)

type logSender struct {
	logger *slog.Logger
}

var _ Sender = (*logSender)(nil)

func newLogSender(logger *slog.Logger) *logSender {
	if logger == nil {
		logger = slog.Default()
	}
	return &logSender{logger: logger}
}

func (s *logSender) Send(_ context.Context, msg Message) error {
	s.logger.LogAttrs(context.Background(), slog.LevelInfo, "email.send",
		slog.String("to", msg.To),
		slog.String("from", msg.From),
		slog.String("subject", msg.Subject),
		slog.Bool("has_html", msg.HTML != ""),
		slog.Bool("has_text", msg.Text != ""),
	)
	return nil
}
