package session

import (
	"cmp"

	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/htmx"
)

// GuardConfig configures the Guard middleware.
type GuardConfig struct {
	// Check tests whether the session is authenticated.
	Check func(*Session) bool

	// RedirectPath is the URL for unauthenticated full-page requests.
	// Defaults to "/signin". Ignored when OnUnauthenticated is set.
	RedirectPath string

	// OnUnauthenticated handles rejected requests. When nil, HTMX requests
	// receive a 401 and full-page requests redirect to RedirectPath with 303.
	OnUnauthenticated func(*web.Context) (web.Response, error)
}

// DefaultGuardConfig returns a GuardConfig with sensible defaults.
func DefaultGuardConfig() GuardConfig {
	return GuardConfig{
		RedirectPath: "/signin",
	}
}

// Guard returns a middleware that rejects unauthenticated requests.
// It panics if the session Middleware is not present in the stack.
func Guard(cfg GuardConfig) web.Middleware {
	check := cfg.Check
	if check == nil {
		panic("web/session: Guard requires GuardConfig.Check")
	}

	redirectPath := cmp.Or(cfg.RedirectPath, "/signin")

	onUnauthed := cfg.OnUnauthenticated
	if onUnauthed == nil {
		onUnauthed = func(c *web.Context) (web.Response, error) {
			if htmx.IsHTMX(c) {
				return web.Response{}, web.ErrUnauthorized("")
			}
			return web.Redirect(303, redirectPath), nil
		}
	}

	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			sess, ok := FromContext(c)
			if !ok {
				panic("web/session: Guard called without session middleware — check middleware stack")
			}
			if !check(sess) {
				return onUnauthed(c)
			}
			return next(c)
		}
	}
}
