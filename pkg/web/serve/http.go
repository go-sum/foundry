// Package adapt bridges web.Handler to net/http, allowing any web.Handler
// to be served by net/http.ListenAndServe or any standard Go HTTP server.
package serve

import (
	"cmp"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-sum/web"
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
	req := FromHTTPRequest(r)
	return web.AcquireContext(r.Context(), req)
}

// FromHTTPRequest converts a standard *http.Request into a web.Request.
//
// Unlike outbound requests, net/http server requests have a path-only r.URL —
// Scheme and Host are always empty. This function builds an absolute URL so
// that middleware (CSRF, OriginGuard) can perform same-origin comparisons without
// relying on X-Forwarded-Proto alone. Scheme precedence: r.TLS (direct TLS)
// → X-Forwarded-Proto header → "http".
func FromHTTPRequest(r *http.Request) web.Request {
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
		switch {
		case r.TLS != nil, strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https"):
			abs.Scheme = "https"
		default:
			abs.Scheme = "http"
		}
	}

	req := web.NewRequest(r.Method, &abs)
	req.Headers = headers
	req.SetBody(r.Body)
	req.SetHost(r.Host)
	req.SetRemoteAddr(r.RemoteAddr)
	return req
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
	return NewContext(r)
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
