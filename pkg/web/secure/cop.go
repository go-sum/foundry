package secure

import (
	"net/http"
	neturl "net/url"
	"strings"

	"github.com/go-sum/web"
)

// COPConfig configures the Cross-Origin Policy middleware.
type COPConfig struct {
	// TrustedOrigins is a list of origins that are allowed in addition to same-origin requests.
	// Example: ["https://cdn.example.com"]
	TrustedOrigins []string

	// TrustedOriginFunc is called for dynamic origin checks. Return true to allow.
	TrustedOriginFunc func(origin string) bool

	// Skipper returns true to bypass COP validation for the request.
	Skipper func(c *web.Context) bool
}

// COP returns middleware that validates cross-origin requests using Fetch Metadata
// (Sec-Fetch-Site) with an Origin/Referer fallback.
//
// Requests with Sec-Fetch-Site: same-origin or none pass unconditionally.
// Requests with Sec-Fetch-Site: cross-site are rejected with 403 unless the
// origin appears in TrustedOrigins or TrustedOriginFunc returns true.
// Safe HTTP methods (GET, HEAD, OPTIONS) are always passed through.
func COP(cfg COPConfig) web.Middleware {
	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			if cfg.Skipper != nil && cfg.Skipper(c) {
				return next(c)
			}

			// Safe methods are read-only; no state change — always permit.
			switch c.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				return next(c)
			}

			// Sec-Fetch-Site is the authoritative source in modern browsers.
			fetchSite := strings.ToLower(strings.TrimSpace(c.Headers.Get("Sec-Fetch-Site")))
			switch fetchSite {
			case "same-origin", "same-site", "none":
				return next(c)
			case "cross-site":
				origin := strings.TrimSpace(c.Headers.Get("Origin"))
				if isTrustedOrigin(origin, cfg) {
					return next(c)
				}
				return web.Response{}, web.ErrForbidden("cross-origin request blocked")
			}

			// Sec-Fetch-Site absent (older browsers) — fall back to Origin/Referer check.
			origin := extractOrigin(c)
			if origin == "" {
				// No origin information: permit (conservative for old browsers).
				return next(c)
			}
			if isSameOriginRequest(c, origin) || isTrustedOrigin(origin, cfg) {
				return next(c)
			}
			return web.Response{}, web.ErrForbidden("cross-origin request blocked")
		}
	}
}

func isTrustedOrigin(origin string, cfg COPConfig) bool {
	for _, trusted := range cfg.TrustedOrigins {
		if sameOrigin(origin, trusted) {
			return true
		}
	}
	if cfg.TrustedOriginFunc != nil {
		return cfg.TrustedOriginFunc(origin)
	}
	return false
}

func isSameOriginRequest(c *web.Context, origin string) bool {
	base := sameOriginBase(c)
	return base != "" && sameOrigin(origin, base)
}

func extractOrigin(c *web.Context) string {
	if origin := strings.TrimSpace(c.Headers.Get("Origin")); origin != "" {
		return origin
	}
	referer := strings.TrimSpace(c.Headers.Get("Referer"))
	if referer == "" {
		return ""
	}
	u, err := neturl.Parse(referer)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
