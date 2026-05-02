package logging

import (
	"io"
	"log/slog"
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

// InitialLoggingConfig returns a Config with sane defaults: info level, text
// format, no explicit output (New will default to os.Stderr).
func InitialLoggingConfig() Config {
	return Config{Level: slog.LevelInfo, Format: FormatText}
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
