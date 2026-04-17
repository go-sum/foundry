package web

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
)

type webContextKey struct{}

// AsyncContext returns a Middleware that stores the current *Context inside the
// underlying context.Context so deep-call code can retrieve it via FromContext
// without explicit parameter plumbing.
//
// It is installed automatically by secure.SecureDefaults() as the first
// middleware in the chain.
func AsyncContext() Middleware {
	return func(next Handler) Handler {
		return func(c *Context) (Response, error) {
			c.ctx = context.WithValue(c.ctx, webContextKey{}, c)
			return next(c)
		}
	}
}

// FromContext retrieves the *Context stored by AsyncContext from any
// non-nil context.Context in the call tree.
//
// Callers should follow the standard library convention and pass a non-nil
// context. FromContext returns nil when AsyncContext was not installed or no
// *Context value is stored in ctx. A nil ctx is tolerated defensively and also
// returns nil.
func FromContext(ctx context.Context) *Context {
	if ctx == nil {
		return nil
	}
	c, _ := ctx.Value(webContextKey{}).(*Context)
	return c
}

// Go runs fn in a new goroutine guarded by a recover. A recovered panic is
// logged at ERROR with event "panic.goroutine", the subsystem label, the
// causal value, and a debug stack trace. After recovery the goroutine exits
// cleanly.
func Go(logger *slog.Logger, subsystem string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if logger == nil {
					logger = slog.Default()
				}
				logger.Error("panic.goroutine",
					slog.String("subsystem", subsystem),
					slog.String("cause", fmt.Sprintf("%v", r)),
					slog.String("stack", string(debug.Stack())),
				)
			}
		}()
		fn()
	}()
}
