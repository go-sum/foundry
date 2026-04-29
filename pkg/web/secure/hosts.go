package secure

import (
	"net"
	"strings"

	"github.com/go-sum/foundry/pkg/web"
)

// AllowedHostsConfig configures the AllowedHosts middleware.
type AllowedHostsConfig struct {
	// Hosts is the set of hostnames (without port) the server legitimately serves.
	// Comparisons are case-insensitive.
	Hosts []string

	// Skipper returns true to bypass host validation for the request.
	Skipper func(c *web.Context) bool
}

// AllowedHosts returns middleware that rejects requests whose Host header does
// not match any entry in cfg.Hosts. The comparison strips the port from the
// request Host and is case-insensitive.
//
// When cfg.Hosts is empty the middleware is a no-op — all requests pass through.
// Requests with an unknown Host receive 421 Misdirected Request.
func AllowedHosts(cfg AllowedHostsConfig) web.Middleware {
	allowed := make(map[string]bool, len(cfg.Hosts))
	for _, h := range cfg.Hosts {
		allowed[strings.ToLower(h)] = true
	}
	return func(next web.Handler) web.Handler {
		if len(allowed) == 0 {
			return next
		}
		return func(c *web.Context) (web.Response, error) {
			if cfg.Skipper != nil && cfg.Skipper(c) {
				return next(c)
			}
			host := c.Request.Host()
			if host == "" {
				return web.Response{}, web.ErrMisdirectedRequest("missing Host header")
			}
			h, _, err := net.SplitHostPort(host)
			if err != nil {
				// Bare IPv6 addresses arrive as "[::1]"; strip brackets so the
				// lookup matches what url.Hostname() produces.
				h = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
			}
			if !allowed[strings.ToLower(h)] {
				return web.Response{}, web.ErrMisdirectedRequest("untrusted Host header")
			}
			return next(c)
		}
	}
}
