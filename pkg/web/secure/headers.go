package secure

import (
	"github.com/go-sum/foundry/pkg/web"
)

// Headers returns middleware that sets security response headers.
// Headers already present in the response are not overwritten (non-clobber semantics),
// so per-route overrides take precedence over global defaults.
func Headers(cfg HeadersConfig) web.Middleware {
	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			resp, err := next(c)
			setIfAbsent(&resp.Headers, "Content-Security-Policy", cfg.ContentSecurityPolicy)
			setIfAbsent(&resp.Headers, "Strict-Transport-Security", cfg.StrictTransportSecurity)
			setIfAbsent(&resp.Headers, "X-Frame-Options", cfg.XFrameOptions)
			setIfAbsent(&resp.Headers, "X-Content-Type-Options", cfg.XContentTypeOptions)
			setIfAbsent(&resp.Headers, "Referrer-Policy", cfg.ReferrerPolicy)
			setIfAbsent(&resp.Headers, "Permissions-Policy", cfg.PermissionsPolicy)
			setIfAbsent(&resp.Headers, "Cross-Origin-Opener-Policy", cfg.CrossOriginOpenerPolicy)
			setIfAbsent(&resp.Headers, "Cross-Origin-Embedder-Policy", cfg.CrossOriginEmbedderPolicy)
			setIfAbsent(&resp.Headers, "Cross-Origin-Resource-Policy", cfg.CrossOriginResourcePolicy)
			return resp, err
		}
	}
}

// setIfAbsent sets header name to value only when the response does not already carry that header.
func setIfAbsent(h *web.Headers, name, value string) {
	if value == "" || h.Has(name) {
		return
	}
	h.Set(name, value)
}
