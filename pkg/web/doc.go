// Package web provides W3C Web API-aligned HTTP primitives for Go servers.
// It models Request, Response, Headers, and related types to mirror the
// shape of the WHATWG Fetch and Web Platform specs, giving handlers a clean,
// composable surface over net/http.
//
// Core:
//   - pkg/web/headers — typed header parsers/serializers (one file per header)
//   - pkg/web/cookiecodec — HMAC-signed and AEAD-encrypted cookie codecs
//   - pkg/web/formdata — streaming multipart and URL-encoded form parsing
//   - pkg/web/file — RFC 7232/7233 conformant file serving with os.Root
//   - pkg/web/validate — struct binding with go-playground/validator
//
// Transport:
//   - pkg/web/router — specificity-sorted pattern router with secure-by-default middleware
//   - pkg/web/serve — net/http bridge and production server configuration
//   - pkg/web/static — static file serving with pre-compressed sidecar support
//   - pkg/web/compress — response compression middleware
//   - pkg/web/etag — ETag generation and conditional response middleware
//   - pkg/web/proxy — HTTP reverse proxy support
//
// Security:
//   - pkg/web/secure — CSRF, CORS, OriginGuard, rate-limit, security headers
//   - pkg/web/session — session management with fixation protection
//   - pkg/web/auth — PKCE, OAuth transaction, SanitizeReturnTo
//
// Resilience:
//   - pkg/web/breaker — circuit breaker
//   - pkg/web/bulkhead — concurrency limiter
//   - pkg/web/retry — retry logic
//   - pkg/web/retrybudget — token-bucket retry budgeting
//   - pkg/web/idempotency — idempotency key middleware
//
// Observability:
//   - pkg/web/logging — request-scoped structured logging
//   - pkg/web/otelweb — OpenTelemetry instrumentation middleware
//
// UI:
//   - pkg/web/htmx — HTMX request detection and response header helpers
//   - pkg/web/render — gomponents-based HTML rendering with SSE and HTMX helpers
//   - pkg/web/site — site identity and origin helpers
//
// Testing:
//   - pkg/web/test — test request builders and response assertions
package web
