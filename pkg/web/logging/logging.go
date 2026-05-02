package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Format controls the log output format.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Config controls the logger factory.
type Config struct {
	Level  slog.Level
	Format Format
	Output io.Writer
}

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

// ParseLogLevel converts a case-insensitive string to a slog.Level.
// Recognises "debug", "warn", and "error". Any other value — including empty
// string — returns slog.LevelInfo.
func ParseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
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
