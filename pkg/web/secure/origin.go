package secure

import (
	"net/http"
	neturl "net/url"
	"strings"

	"github.com/go-sum/web"
)

// OriginGuardConfig configures the OriginGuard middleware.
type OriginGuardConfig struct {
	// TrustedOrigins is a list of origins allowed in addition to same-origin
	// requests. Example: ["https://cdn.example.com"]
	TrustedOrigins []string

	// TrustedOriginFunc is called for dynamic origin checks. Return true to allow.
	TrustedOriginFunc func(origin string) bool

	// PermitUnknownOrigin allows requests that carry no origin information
	// (absent Sec-Fetch-Site, Origin, and Referer). By default such requests
	// are rejected. Set this to true to permit them, which is more permissive
	// for older browsers that do not send Fetch Metadata headers.
	PermitUnknownOrigin bool

	// Skipper returns true to bypass origin validation for the request.
	Skipper func(c *web.Context) bool
}

// OriginGuard returns middleware that validates cross-origin requests using
// Fetch Metadata (Sec-Fetch-Site) with an Origin/Referer fallback.
//
// By default the middleware is strict: requests with no origin information
// are rejected. Set PermitUnknownOrigin: true to allow them (conservative
// mode for older browsers that omit Fetch Metadata headers).
//
// Safe HTTP methods (GET, HEAD, OPTIONS) always pass through.
// Requests with Sec-Fetch-Site: same-origin, same-site, or none pass unconditionally.
// Requests with Sec-Fetch-Site: cross-site are rejected with 403 unless the
// origin appears in TrustedOrigins or TrustedOriginFunc returns true.
func OriginGuard(cfg OriginGuardConfig) web.Middleware {
	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			if cfg.Skipper != nil && cfg.Skipper(c) {
				return next(c)
			}

			// Safe methods are read-only; no state change — always permit.
			switch c.Method() {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				return next(c)
			}

			// Sec-Fetch-Site is the authoritative source in modern browsers.
			fetchSite := strings.ToLower(strings.TrimSpace(c.Headers().Get("Sec-Fetch-Site")))
			switch fetchSite {
			case "same-origin", "same-site", "none":
				return next(c)
			case "cross-site":
				origin := strings.TrimSpace(c.Headers().Get("Origin"))
				if isTrustedOrigin(origin, cfg) {
					return next(c)
				}
				return web.Response{}, web.ErrForbidden("cross-origin request blocked")
			}

			// Sec-Fetch-Site absent — fall back to Origin/Referer.
			origin := extractOrigin(c)
			if origin == "" {
				if cfg.PermitUnknownOrigin {
					return next(c)
				}
				return web.Response{}, web.ErrForbidden("cross-origin request blocked")
			}
			if isSameOriginRequest(c, origin) || isTrustedOrigin(origin, cfg) {
				return next(c)
			}
			return web.Response{}, web.ErrForbidden("cross-origin request blocked")
		}
	}
}

func isTrustedOrigin(origin string, cfg OriginGuardConfig) bool {
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
	if origin := strings.TrimSpace(c.Headers().Get("Origin")); origin != "" {
		return origin
	}
	referer := strings.TrimSpace(c.Headers().Get("Referer"))
	if referer == "" {
		return ""
	}
	u, err := neturl.Parse(referer)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
