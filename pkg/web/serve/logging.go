package serve

import (
	"cmp"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-sum/web"
)

// AccessLogMiddleware returns middleware that emits a structured slog log entry
// for each request with: method, path, status, latency (duration), request_id.
// It must be placed after web.WithRequestID() to capture the request ID.
// Logs at Warn for status >= 400, Error for status >= 500, Info otherwise.
func AccessLogMiddleware() web.Middleware {
	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			start := time.Now()
			resp, err := next(c)
			latency := time.Since(start)

			path := ""
			if c.URL != nil {
				path = c.URL.Path
			}

			status := cmp.Or(resp.Status, http.StatusOK)

			level := slog.LevelInfo
			if status >= 500 {
				level = slog.LevelError
			} else if status >= 400 {
				level = slog.LevelWarn
			}

			slog.LogAttrs(c.Context(), level, "http.request",
				slog.String("method", c.Method),
				slog.String("path", path),
				slog.Int("status", status),
				slog.Int64("latency_ms", latency.Milliseconds()),
				slog.String("request_id", web.RequestID(c)),
			)

			return resp, err
		}
	}
}
