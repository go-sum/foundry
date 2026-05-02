package secure

import (
	"net/http"
	"time"

	"github.com/go-sum/foundry/pkg/web"
)

// DefaultCSPTemplate is the default Content-Security-Policy template. It includes
// 'nonce-{nonce}' placeholders that CSPNonce replaces with a fresh per-request value.
const DefaultCSPTemplate = "default-src 'self'; script-src 'self' 'nonce-{nonce}'; style-src 'self' 'nonce-{nonce}'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; object-src 'none'"

// CSPNonceConfig configures per-request Content-Security-Policy nonce injection.
type CSPNonceConfig struct {
	// CSPTemplate is the Content-Security-Policy header value template.
	// The literal string "{nonce}" is replaced with a freshly generated
	// base64url-encoded nonce on every request.
	//
	// Example:
	//   "default-src 'self'; script-src 'self' 'nonce-{nonce}'; style-src 'self' 'nonce-{nonce}'"
	//
	// When CSPTemplate is empty, no Content-Security-Policy header is set.
	//
	// Composition note: when using CSPNonce alongside the Headers middleware,
	// set HeadersConfig.ContentSecurityPolicy to "" so the two middlewares
	// do not conflict. CSPNonce must be placed AFTER Headers in the chain
	// (outermost middleware runs first, so list CSPNonce before Headers when
	// calling router.Use or web.Chain).
	CSPTemplate string

	// ScriptSrcExtra holds additional script-src tokens (e.g. CSP hashes for
	// inline scripts) that are appended to the script-src directive at
	// middleware init time. Each entry should be a complete CSP source
	// expression such as "'sha256-abc123...'" .
	ScriptSrcExtra []string
}

// WithScriptHashes returns a copy of cfg with the given CSP hash tokens
// appended to ScriptSrcExtra.
func (cfg CSPNonceConfig) WithScriptHashes(hashes ...string) CSPNonceConfig {
	cfg.ScriptSrcExtra = append(cfg.ScriptSrcExtra, hashes...)
	return cfg
}

// InitialCSPNonceConfig returns a CSPNonceConfig using DefaultCSPTemplate.
func InitialCSPNonceConfig() CSPNonceConfig {
	return CSPNonceConfig{CSPTemplate: DefaultCSPTemplate}
}

// HeadersConfig configures security headers middleware.
type HeadersConfig struct {
	// ContentSecurityPolicy sets the Content-Security-Policy header.
	// Integrate with CSPNonce middleware to inject nonce into script-src/style-src.
	ContentSecurityPolicy string

	// StrictTransportSecurity sets the Strict-Transport-Security header.
	// Defaults to 2-year max-age with preload when using InitialHeadersConfig.
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

// InitialHeadersConfig returns a strict, production-ready security headers configuration.
// All values follow current best-practice recommendations:
//   - HSTS with 2-year max-age, includeSubDomains, and preload
//   - COOP same-origin to isolate the browsing context
//   - COEP require-corp to enable cross-origin isolation
//   - CORP same-origin
//   - CSP with frame-ancestors 'none', base-uri 'self', form-action 'self'
//   - Permissions-Policy denying all features by default
//   - Strict Referrer-Policy
func InitialHeadersConfig() HeadersConfig {
	return HeadersConfig{
		ContentSecurityPolicy:     "default-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; object-src 'none'",
		StrictTransportSecurity:   "max-age=63072000; includeSubDomains; preload",
		XFrameOptions:             "DENY",
		XContentTypeOptions:       "nosniff",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		PermissionsPolicy:         "accelerometer=(), autoplay=(), camera=(), cross-origin-isolated=(), display-capture=(), encrypted-media=(), fullscreen=(), geolocation=(), gyroscope=(), keyboard-map=(), magnetometer=(), microphone=(), midi=(), payment=(), picture-in-picture=(), publickey-credentials-get=(), screen-wake-lock=(), sync-xhr=(), usb=(), web-share=(), xr-spatial-tracking=()",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginResourcePolicy: "same-origin",
	}
}

// CSRFConfig configures CSRF protection middleware.
type CSRFConfig struct {
	// Key is the HMAC-SHA256 signing key for stateless fallback mode.
	Key []byte `validate:"required,min=32" help:"set SECURITY_CSRF_KEY — generate with 'openssl rand -hex 32' and place in starter/.env"`

	// TokenTTL is the stateless token lifetime.
	TokenTTL time.Duration

	// ContextKey is the context/session key under which the token is stored.
	// Defaults to "csrf".
	ContextKey string

	// HeaderName is the header name checked for the token.
	// Defaults to "X-CSRF-Token".
	HeaderName string

	// FormField is the form body field checked for the token on
	// application/x-www-form-urlencoded and multipart/form-data requests.
	// The body is peeked via Clone so the downstream handler still receives
	// the full original body. Defaults to "_csrf".
	// This matches the field name emitted by render.CSRFField.
	FormField string

	// SafeMethods are HTTP methods that do not require CSRF validation.
	// Defaults to GET, HEAD, OPTIONS.
	SafeMethods []string

	// AllowMissingOrigin controls whether unsafe requests without Origin or
	// Referer are allowed. Defaults to false — requests that omit both headers
	// are rejected. Set to true only in development or when clients are known
	// to omit origin headers (e.g. some native HTTP clients).
	AllowMissingOrigin bool

	// AllowedOrigins are additional trusted origins for unsafe requests.
	AllowedOrigins []string

	// AllowedOriginFunc performs dynamic origin checks for unsafe requests.
	AllowedOriginFunc func(origin string, c *web.Context) bool

	// ServerOrigin is the server-owned origin (e.g. "https://example.com")
	// used for same-origin comparison instead of deriving it from the request
	// Host header. When empty, falls back to the request-derived origin.
	ServerOrigin string

	// TokenLookup overrides default submitted-token extraction.
	TokenLookup func(c *web.Context) string

	// Skipper returns true to skip CSRF validation for the request.
	Skipper func(c *web.Context) bool

	// CookieName is the name of the double-submit CSRF cookie used in
	// stateless mode. Defaults to "csrf". For production over HTTPS, use
	// "__Host-csrf" with CookieSecure set to true.
	CookieName string

	// CookieSecure sets the Secure attribute on the stateless CSRF cookie.
	// Set to true in production (HTTPS-only). Defaults to false.
	CookieSecure bool

	// CookieSameSite controls the SameSite attribute of the CSRF cookie.
	// Accepted values: "Lax" (default), "Strict", "None". Empty string defaults to "Lax".
	CookieSameSite string
}

// InitialCSRFConfig returns a CSRFConfig with production-ready defaults applied.
// Key is zero-length; supply it via CSRFConfigFromHex. CookieSecure defaults to
// true — set it to false in testing overlays (plain HTTP). AllowMissingOrigin
// defaults to false; set it to true only in testing where httptest omits Origin.
func InitialCSRFConfig() CSRFConfig {
	return CSRFConfig{
		TokenTTL:     time.Hour,
		ContextKey:   "csrf",
		HeaderName:   "X-CSRF-Token",
		FormField:    "_csrf",
		CookieName:   "csrf",
		CookieSecure: true,
		SafeMethods: []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodOptions,
		},
	}
}

// SecureConfig is a composite configuration for all security middleware.
type SecureConfig struct {
	CSP     CSPNonceConfig
	CSRF    CSRFConfig
	Headers HeadersConfig
}

// InitialSecureConfig returns a SecureConfig with production-ready defaults
// for all security middleware components.
func InitialSecureConfig() SecureConfig {
	return SecureConfig{
		CSP:     InitialCSPNonceConfig(),
		CSRF:    InitialCSRFConfig(),
		Headers: InitialHeadersConfig(),
	}
}
