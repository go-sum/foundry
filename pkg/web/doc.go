// Package web provides W3C Web API-aligned HTTP primitives for Go servers.
// It models Request, Response, Headers, and related types to mirror the
// shape of the WHATWG Fetch and Web Platform specs, giving handlers a clean,
// composable surface over net/http.
//
// Key packages:
//   - pkg/web/headers — typed header parsers/serializers (one file per header)
//   - pkg/web/cookiecodec — HMAC-signed and AEAD-encrypted cookie codecs
//   - pkg/web/formdata — streaming multipart and URL-encoded form parsing
//   - pkg/web/file — RFC 7232/7233 conformant file serving with os.Root
//   - pkg/web/session — session management with fixation protection
//   - pkg/web/secure — CSRF, CORS, COP, rate-limit, security headers
//   - pkg/web/htmx — HTMX request detection and response header helpers
//   - pkg/web/adapt — net/http bridge and production server configuration
//   - pkg/web/auth — PKCE, OAuth transaction, SanitizeReturnTo
//   - pkg/web/router — specificity-sorted pattern router with secure-by-default middleware
package web
