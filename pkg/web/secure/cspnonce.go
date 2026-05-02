package secure

import (
	"crypto/rand"
	"encoding/base64"
	"strings"

	"github.com/go-sum/foundry/pkg/web"
)

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
	tmpl := cfg.CSPTemplate
	if len(cfg.ScriptSrcExtra) > 0 && tmpl != "" {
		extra := " " + strings.Join(cfg.ScriptSrcExtra, " ")
		tmpl = strings.Replace(tmpl, "; style-src", extra+"; style-src", 1)
	}

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

			if tmpl != "" {
				policy := strings.ReplaceAll(tmpl, "{nonce}", nonce)
				resp.Headers.Set("Content-Security-Policy", policy)
			}

			return resp, err
		}
	}
}
