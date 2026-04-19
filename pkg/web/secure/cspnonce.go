package secure

import (
	"crypto/rand"
	"encoding/base64"
	"strings"

	"github.com/go-sum/web"
)

// DefaultCSPTemplate is the default Content-Security-Policy template. It includes
// 'nonce-{nonce}' placeholders that CSPNonce replaces with a fresh per-request value.
const DefaultCSPTemplate = "default-src 'self'; script-src 'self' 'nonce-{nonce}'; style-src 'self' 'nonce-{nonce}'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; object-src 'none'"

// DefaultCSPNonceConfig returns a CSPNonceConfig using DefaultCSPTemplate.
func DefaultCSPNonceConfig() CSPNonceConfig {
	return CSPNonceConfig{CSPTemplate: DefaultCSPTemplate}
}

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
}

type nonceContextKey struct{}

// Nonce retrieves the CSP nonce from c. Returns "" if no nonce is present.
func Nonce(c *web.Context) string {
	n, _ := web.Get[string](c, nonceContextKey{})
	return n
}

// CSPNonce returns middleware that generates a cryptographically random nonce
// per request, stores it in ctx under nonceContextKey{}, and sets the
// Content-Security-Policy response header with "{nonce}" replaced by the
// generated value.
//
// The nonce is 16 random bytes encoded as base64url (no padding), producing
// a 22-character string. This satisfies the minimum entropy recommended by
// the CSP3 spec (128 bits).
func CSPNonce(cfg CSPNonceConfig) web.Middleware {
	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			// Generate 16 random bytes → base64url, no padding.
			var raw [16]byte
			if _, err := rand.Read(raw[:]); err != nil {
				return web.Response{}, web.ErrInternal(err)
			}
			nonce := base64.RawURLEncoding.EncodeToString(raw[:])

			c.Set(nonceContextKey{}, nonce)
			resp, err := next(c)

			if cfg.CSPTemplate != "" {
				policy := strings.ReplaceAll(cfg.CSPTemplate, "{nonce}", nonce)
				resp.Headers.Set("Content-Security-Policy", policy)
			}

			return resp, err
		}
	}
}
