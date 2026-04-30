// Package health provides a generic HTTP health check handler and checker types
// for production readiness endpoints (e.g. /healthz).
package health

import (
	"context"
	"net/http"

	"github.com/go-sum/foundry/pkg/web"
)

// Checker reports the health of a single dependency.
type Checker interface {
	Check(ctx context.Context) error
}

// CheckerFunc is a function that implements Checker.
type CheckerFunc func(ctx context.Context) error

func (f CheckerFunc) Check(ctx context.Context) error { return f(ctx) }

// DBChecker wraps any value with a Ping(context.Context) error method as a Checker.
// Suitable for *pgxpool.Pool and similar connection pool types.
func DBChecker(pinger interface{ Ping(context.Context) error }) Checker {
	return CheckerFunc(func(ctx context.Context) error { return pinger.Ping(ctx) })
}

// KVChecker wraps any value with a Ping(context.Context) error method as a Checker.
// Suitable for kv.Store implementations.
func KVChecker(pinger interface{ Ping(context.Context) error }) Checker {
	return CheckerFunc(func(ctx context.Context) error { return pinger.Ping(ctx) })
}

// Handler returns a web.Handler that runs all checkers. Returns 200 OK with body "ok"
// if all checkers pass, or a 503 ServiceUnavailable error on any failure.
func Handler(checkers ...Checker) web.Handler {
	return func(c *web.Context) (web.Response, error) {
		for _, ch := range checkers {
			if err := ch.Check(c.Context()); err != nil {
				return web.Response{}, web.ErrUnavailable("service unhealthy", err)
			}
		}
		return web.Text(http.StatusOK, "ok"), nil
	}
}
