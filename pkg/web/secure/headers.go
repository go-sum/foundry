package secure

import (
	"github.com/go-sum/web"
)

// HeadersConfig configures security headers middleware.
type HeadersConfig struct {
	// ContentSecurityPolicy sets the Content-Security-Policy header.
	// Integrate with CSPNonce middleware to inject nonce into script-src/style-src.
	ContentSecurityPolicy string

	// StrictTransportSecurity sets the Strict-Transport-Security header.
	// Defaults to 2-year max-age with preload when using DefaultHeadersConfig.
	StrictTransportSecurity string

	// XFrameOptions sets the X-Frame-Options header.
	XFrameOptions string

	// XContentTypeOptions sets the X-Content-Type-Options header.
	XContentTypeOptions string

	// ReferrerPolicy sets the Referrer-Policy header.
	ReferrerPolicy string

	// PermissionsPolicy sets the Permissions-Policy header.
	PermissionsPolicy string

	// CrossOriginOpenerPolicy sets the Cross-Origin-Opener-Policy header.
	CrossOriginOpenerPolicy string

	// CrossOriginEmbedderPolicy sets the Cross-Origin-Embedder-Policy header.
	CrossOriginEmbedderPolicy string

	// CrossOriginResourcePolicy sets the Cross-Origin-Resource-Policy header.
	CrossOriginResourcePolicy string
}

// DefaultHeadersConfig returns a strict, production-ready security headers configuration.
// All values follow current best-practice recommendations:
//   - HSTS with 2-year max-age, includeSubDomains, and preload
//   - COOP same-origin to isolate the browsing context
//   - COEP require-corp to enable cross-origin isolation
//   - CORP same-origin
//   - CSP with frame-ancestors 'none', base-uri 'self', form-action 'self'
//   - Permissions-Policy denying all features by default
//   - Strict Referrer-Policy
func DefaultHeadersConfig() HeadersConfig {
	return HeadersConfig{
		ContentSecurityPolicy:     "default-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; object-src 'none'",
		StrictTransportSecurity:   "max-age=63072000; includeSubDomains; preload",
		XFrameOptions:             "DENY",
		XContentTypeOptions:       "nosniff",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		PermissionsPolicy:         "accelerometer=(), ambient-light-sensor=(), autoplay=(), battery=(), camera=(), cross-origin-isolated=(), display-capture=(), document-domain=(), encrypted-media=(), execution-while-not-rendered=(), execution-while-out-of-viewport=(), fullscreen=(), geolocation=(), gyroscope=(), keyboard-map=(), magnetometer=(), microphone=(), midi=(), navigation-override=(), payment=(), picture-in-picture=(), publickey-credentials-get=(), screen-wake-lock=(), sync-xhr=(), usb=(), web-share=(), xr-spatial-tracking=()",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginResourcePolicy: "same-origin",
	}
}

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
