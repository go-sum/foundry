// Package adapt bridges web.Handler to net/http, allowing any web.Handler
// to be served by net/http.ListenAndServe or any standard Go HTTP server.
package serve

import (
	"cmp"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-sum/foundry/pkg/web"
)

// defaultMaxRequestBodyBytes is the default body size ceiling applied by
// ToHTTPHandler when no explicit limit is configured. Callers may raise this
// limit via Config.MaxRequestBodyBytes.
const defaultMaxRequestBodyBytes = 32 * 1024 * 1024 // 32 MiB

// Config configures the HTTP adapter.
type Config struct {
	// MaxRequestBodyBytes limits the size of incoming request bodies.
	// Requests exceeding this limit will cause body reads to return an error
	// wrapping web.ErrBodyTooLarge. Set to 0 for no limit (not recommended).
	// A sensible default is 32 MiB (32 << 20).
	MaxRequestBodyBytes int64

	// OnError is called when writing the response body fails. Defaults to
	// logging the error at ERROR level via slog.Default().
	OnError func(err error)

	// TrustedProxies lists parsed CIDR ranges whose X-Forwarded-Proto header
	// is accepted for scheme detection. When empty, scheme is inferred from
	// TLS state only. Build this slice via ParseTrustedProxies.
	TrustedProxies []*net.IPNet
}

// ToHTTPHandler wraps a web.Handler as an http.Handler with default config.
func ToHTTPHandler(h web.Handler) http.Handler {
	return ToHTTPHandlerWithConfig(h, Config{})
}

// ToHTTPHandlerWithConfig wraps a web.Handler as an http.Handler with the
// given configuration.
func ToHTTPHandlerWithConfig(h web.Handler, cfg Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := acquireContextWithConfig(w, r, cfg)
		defer web.ReleaseContext(c)
		resp, err := h(c)
		if err != nil {
			// Fallback path: ErrorBoundary was not in the chain. Classify the
			// error and synthesize a problem+json response so the adapter
			// always writes a valid response.
			e := web.Classify(err)
			resp = web.Problem(c, e)
		}
		WriteHTTPResponse(w, r, resp, cfg)
	})
}

// acquireContextWithConfig applies the body size limit from cfg and returns a
// pooled *web.Context. The caller must defer web.ReleaseContext on the result.
func acquireContextWithConfig(w http.ResponseWriter, r *http.Request, cfg Config) *web.Context {
	if r.Body != nil {
		limit := cmp.Or(cfg.MaxRequestBodyBytes, defaultMaxRequestBodyBytes)
		r.Body = http.MaxBytesReader(w, r.Body, limit)
	}
	req := fromHTTPRequestWithConfig(r, cfg)
	return web.AcquireContext(r.Context(), req)
}

// FromHTTPRequest converts a standard *http.Request into a web.Request.
//
// Unlike outbound requests, net/http server requests have a path-only r.URL —
// Scheme and Host are always empty. This function builds an absolute URL so
// that middleware (CSRF, OriginGuard) can perform same-origin comparisons.
// Scheme precedence: r.TLS (direct TLS) → "http". X-Forwarded-Proto is never
// trusted here; use ToHTTPHandlerWithConfig with Config.TrustedProxies to
// accept forwarded-proto from known proxy peers.
func FromHTTPRequest(r *http.Request) web.Request {
	return buildWebRequest(r, nil)
}

// fromHTTPRequestWithConfig is like FromHTTPRequest but gates X-Forwarded-Proto
// trust on cfg.TrustedProxies.
func fromHTTPRequestWithConfig(r *http.Request, cfg Config) web.Request {
	return buildWebRequest(r, cfg.TrustedProxies)
}

// buildWebRequest is the shared implementation behind FromHTTPRequest and
// fromHTTPRequestWithConfig.
func buildWebRequest(r *http.Request, trusted []*net.IPNet) web.Request {
	headers := web.NewHeaders()
	for name, values := range r.Header {
		for _, v := range values {
			headers.Append(name, v)
		}
	}

	base := r.URL
	if base == nil {
		base = &url.URL{}
	}

	// Build an absolute URL: copy the path/query from r.URL, then populate
	// Scheme and Host which are always empty for server-side requests.
	abs := *base
	if abs.Host == "" {
		abs.Host = r.Host
	}
	if abs.Scheme == "" {
		abs.Scheme = resolveScheme(r, trusted)
	}

	req := web.NewRequest(r.Method, &abs)
	req.Headers = headers
	req.SetBody(r.Body)
	req.SetHost(r.Host)
	req.SetRemoteAddr(r.RemoteAddr)
	return req
}

// resolveScheme determines the request scheme. Direct TLS always wins. When
// TLS is absent, X-Forwarded-Proto is accepted only from a trusted peer and
// only when the header is present exactly once, contains no commas, and is
// exactly "http" or "https". Duplicate, comma-separated, or malformed values
// are ignored and the default "http" is returned.
func resolveScheme(r *http.Request, trusted []*net.IPNet) string {
	if r.TLS != nil {
		return "https"
	}
	if !IsTrustedProxy(r.RemoteAddr, trusted) {
		return "http"
	}
	vals := r.Header.Values("X-Forwarded-Proto")
	if len(vals) != 1 {
		return "http"
	}
	if strings.Contains(vals[0], ",") {
		return "http"
	}
	p := strings.ToLower(strings.TrimSpace(vals[0]))
	if p == "http" || p == "https" {
		return p
	}
	return "http"
}

// NormalizeProxyIP canonicalizes an IP token from RemoteAddr, X-Forwarded-For,
// or Forwarded. Accepts bare IPs, bracketed IPv6 literals, and host:port forms.
// Returns the canonical Go net.IP string and true on success.
func NormalizeProxyIP(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if parsed := net.ParseIP(strings.Trim(raw, "[]")); parsed != nil {
		return parsed.String(), true
	}
	if splitHost, _, err := net.SplitHostPort(raw); err == nil {
		raw = splitHost
	}
	raw = strings.Trim(strings.TrimSpace(raw), "[]")
	parsed := net.ParseIP(raw)
	if parsed == nil {
		return "", false
	}
	return parsed.String(), true
}

// IsTrustedProxy reports whether remoteAddr falls within any of the trusted CIDRs.
// It accepts bare IPs, bracketed IPv6, host:port, and IPv4-mapped IPv6 forms.
func IsTrustedProxy(remoteAddr string, trusted []*net.IPNet) bool {
	if len(trusted) == 0 {
		return false
	}
	normalized, ok := NormalizeProxyIP(remoteAddr)
	if !ok {
		return false
	}
	ip := net.ParseIP(normalized)
	for _, cidr := range trusted {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// ParseTrustedProxies parses CIDR strings into []*net.IPNet for use in
// Config.TrustedProxies. Returns an error for the first unparseable entry.
func ParseTrustedProxies(cidrs []string) ([]*net.IPNet, error) {
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, s := range cidrs {
		_, ipnet, err := net.ParseCIDR(s)
		if err != nil {
			return nil, fmt.Errorf("web/serve: invalid trusted proxy CIDR %q: %w", s, err)
		}
		out = append(out, ipnet)
	}
	return out, nil
}

// NewContext converts an *http.Request into a *web.Context without adapter config.
func NewContext(r *http.Request) *web.Context {
	return web.NewContext(r.Context(), FromHTTPRequest(r))
}

// NewContextWithConfig converts an *http.Request into a *web.Context, applying adapter config.
func NewContextWithConfig(w http.ResponseWriter, r *http.Request, cfg Config) *web.Context {
	if r.Body != nil {
		limit := cmp.Or(cfg.MaxRequestBodyBytes, defaultMaxRequestBodyBytes)
		r.Body = http.MaxBytesReader(w, r.Body, limit)
	}
	return web.NewContext(r.Context(), fromHTTPRequestWithConfig(r, cfg))
}

// WriteHTTPResponse writes a web.Response back to net/http.
func WriteHTTPResponse(w http.ResponseWriter, r *http.Request, resp web.Response, cfg Config) {
	onError := cfg.OnError
	if onError == nil {
		onError = func(err error) {
			slog.Error("adapt: response body write error", "error", err)
		}
	}

	warn := func(message string, args ...any) {
		if cfg.OnError != nil {
			onError(fmt.Errorf("%s: %v", message, args))
			return
		}
		slog.Warn(message, args...)
	}

	resp.Headers.ForEach(func(name string, values []string) {
		if web.IsForbiddenResponseHeaderForStatus(name, resp.Status) {
			warn("adapt: dropping transport-controlled response header", "header", name)
			return
		}
		for _, v := range values {
			w.Header().Add(name, v)
		}
	})

	if resp.Status == 0 {
		resp.Status = http.StatusOK
	}
	w.WriteHeader(resp.Status)

	// Handle 101 Switching Protocols (WebSocket) — hijack the connection.
	if resp.Status == http.StatusSwitchingProtocols {
		if hb, ok := resp.Body.(*hijackBody); ok {
			defer func() { _ = resp.Body.Close() }()
			if hb.fn == nil {
				onError(fmt.Errorf("adapt: Switching called with nil HijackFunc"))
				return
			}
			rc := http.NewResponseController(w)
			conn, brw, err := rc.Hijack()
			if err != nil {
				onError(fmt.Errorf("adapt: hijack failed: %w", err))
				return
			}
			if err := hb.fn(conn, brw); err != nil {
				onError(fmt.Errorf("adapt: websocket handler error: %w", err))
			}
			return
		}
	}

	if resp.Body == nil {
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			onError(err)
		}
	}()

	if r != nil && r.Method == http.MethodHead {
		return
	}

	if _, err := io.Copy(w, resp.Body); err != nil {
		onError(err)
	}
}
