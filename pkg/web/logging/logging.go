package logging

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey struct{}

// New builds a *slog.Logger from cfg. A nil Output defaults to os.Stderr.
// An empty Format defaults to FormatText.
func New(cfg Config) *slog.Logger {
	out := cfg.Output
	if out == nil {
		out = os.Stderr
	}
	opts := &slog.HandlerOptions{Level: cfg.Level}
	var handler slog.Handler
	if cfg.Format == FormatJSON {
		handler = slog.NewJSONHandler(out, opts)
	} else {
		handler = slog.NewTextHandler(out, opts)
	}
	return slog.New(handler)
}

// IntoContext stores l in ctx and returns the derived context.
func IntoContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext retrieves the logger stored by IntoContext. Returns slog.Default()
// if no logger is present.
func FromContext(ctx context.Context) *slog.Logger {
	l, ok := ctx.Value(ctxKey{}).(*slog.Logger)
	if !ok || l == nil {
		return slog.Default()
	}
	return l
}

// WithRequestID returns l with a "request_id" attribute added. Returns l
// unchanged if id is empty. A nil l defaults to slog.Default().
func WithRequestID(l *slog.Logger, id string) *slog.Logger {
	if l == nil {
		l = slog.Default()
	}
	if id == "" {
		return l
	}
	return l.With("request_id", id)
}
