package logging

import (
	"log/slog"

	"github.com/go-sum/foundry/pkg/web"
)

// Middleware returns a web.Middleware that attaches a request-scoped logger
// (enriched with the request ID) into the request context.
func Middleware(l *slog.Logger) web.Middleware {
	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			reqLogger := WithRequestID(l, web.RequestID(c))
			c.SetContext(IntoContext(c.Context(), reqLogger))
			return next(c)
		}
	}
}
