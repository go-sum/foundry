package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
)

// BoundaryConfig configures the ErrorBoundary middleware.
type BoundaryConfig struct {
	// Renderer is called for HTML/HTMX responses. When nil, Problem is used for
	// all responses.
	Renderer ErrorRenderer

	// Logger is the structured logger used for error events. When nil,
	// slog.Default() is used.
	Logger *slog.Logger

	// OnPanic is invoked when a handler panics. It receives the panic value and
	// the captured stack trace. May be nil.
	OnPanic func(any, []byte)

	// TypeBase overrides DefaultTypeBase when non-empty. Applied once at
	// middleware construction time.
	TypeBase string

	// CaptureStack enables stack trace capture for non-panic 5xx errors.
	// When true, the stack is included in the http.error log event under a
	// "stack" attribute and is never forwarded to the client.
	CaptureStack bool

	// Op, Subsystem, TraceID, SpanID, DedupeKey are optional per-request
	// extractors. A nil extractor omits the corresponding field from the http.error
	// log event. They are called with the current *Context on every error event.
	Op        func(*Context) string
	Subsystem func(*Context) string
	TraceID   func(*Context) string
	SpanID    func(*Context) string
	DedupeKey func(*Context) string

	// OnError is an optional hook invoked after error classification and logging,
	// but before rendering. It receives the request context and the classified
	// *Error. Use it to record errors on observability backends (e.g., OTel spans).
	// May be nil.
	OnError func(ctx context.Context, e *Error)
}

// ErrorBoundary returns a Middleware that:
//  1. Recovers from panics, classifies them as 500, logs with stack, and renders.
//  2. Classifies non-nil errors returned by the inner handler via Classify.
//  3. Logs at Debug for 499/client-canceled, Warn for other 4xx, Error for 5xx.
//  4. Renders via cfg.Renderer when the client prefers HTML/HTMX, otherwise
//     via Problem.
//  5. Strips the body on HEAD requests.
//  6. Always returns (Response, nil) — errors are consumed and rendered.
func ErrorBoundary(cfg BoundaryConfig) Middleware {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	typeBase := cfg.TypeBase

	return func(next Handler) Handler {
		return func(c *Context) (resp Response, herr error) {
			defer func() {
				r := recover()
				if r == nil {
					return
				}
				stack := debug.Stack()
				if cfg.OnPanic != nil {
					cfg.OnPanic(r, stack)
				}
				panicErr := fmt.Errorf("panic: %v", r)
				e := ErrInternal(panicErr)

				attrs := []slog.Attr{
					slog.Int("status", e.Status),
					slog.String("code", string(e.Code)),
					slog.String("request_id", RequestID(c)),
					slog.String("cause", panicErr.Error()),
					slog.String("stack", string(stack)),
				}
				attrs = appendIfExtracted(attrs, "op", cfg.Op, c)
				attrs = appendIfExtracted(attrs, "subsystem", cfg.Subsystem, c)
				attrs = appendIfExtracted(attrs, "trace_id", cfg.TraceID, c)
				attrs = appendIfExtracted(attrs, "span_id", cfg.SpanID, c)
				attrs = appendIfExtracted(attrs, "dedupe_key", cfg.DedupeKey, c)
				logger.LogAttrs(c.Context(), slog.LevelError, "http.error", attrs...)

				if cfg.OnError != nil {
					cfg.OnError(c.Context(), e)
				}
				resp = renderError(c, cfg.Renderer, typeBase, e)
				if c.Method() == http.MethodHead {
					resp.Body = nil
				}
				herr = nil // boundary consumed the error
			}()

			resp, herr = next(c)
			if herr == nil {
				return resp, nil
			}

			e := Classify(herr)

			level := slog.LevelWarn
			switch {
			case errors.Is(herr, context.Canceled):
				level = slog.LevelDebug
			case e.Status == 499:
				level = slog.LevelDebug
			case e.Status >= 500:
				level = slog.LevelError
			}

			attrs := []slog.Attr{
				slog.Int("status", e.Status),
				slog.String("code", string(e.Code)),
				slog.String("request_id", RequestID(c)),
				slog.String("cause", fmt.Sprintf("%v", e.Cause)),
			}
			if cfg.CaptureStack && e.Status >= 500 {
				stackAttr := slog.String("stack", string(debug.Stack()))
				if stackAttr.Key != "" {
					attrs = append(attrs, stackAttr)
				}
			}
			attrs = appendIfExtracted(attrs, "op", cfg.Op, c)
			attrs = appendIfExtracted(attrs, "subsystem", cfg.Subsystem, c)
			attrs = appendIfExtracted(attrs, "trace_id", cfg.TraceID, c)
			attrs = appendIfExtracted(attrs, "span_id", cfg.SpanID, c)
			attrs = appendIfExtracted(attrs, "dedupe_key", cfg.DedupeKey, c)
			logger.LogAttrs(c.Context(), level, "http.error", attrs...)

			if cfg.OnError != nil {
				cfg.OnError(c.Context(), e)
			}
			resp = renderError(c, cfg.Renderer, typeBase, e)
			if c.Method() == http.MethodHead {
				resp.Body = nil
			}
			return resp, nil // boundary consumed the error
		}
	}
}

// appendIfExtracted appends a slog.String attr to attrs if f is non-nil and
// returns a non-empty string for the given context.
func appendIfExtracted(attrs []slog.Attr, key string, f func(*Context) string, c *Context) []slog.Attr {
	if f == nil {
		return attrs
	}
	if v := f(c); v != "" {
		return append(attrs, slog.String(key, v))
	}
	return attrs
}

// renderError chooses between HTML rendering and problem+json. typeBase
// overrides DefaultTypeBase for this boundary instance without mutating globals.
func renderError(c *Context, renderer ErrorRenderer, typeBase string, e *Error) Response {
	if e.TypeURI == "" && typeBase != "" {
		e.TypeURI = typeBase
	}
	var resp Response
	if renderer != nil && preferHTML(c) {
		resp = renderer.RenderError(c, e)
	} else {
		resp = Problem(c, e)
	}
	for name, value := range e.ResponseHeaders {
		resp.Headers.Set(name, value)
	}
	return resp
}

// preferHTML reports whether the client prefers an HTML response based on the
// HTMX request header and the Accept header.
func preferHTML(c *Context) bool {
	if c.Headers().Get("HX-Request") == "true" {
		return true
	}
	accept := c.Headers().Get("Accept")
	return strings.Contains(accept, "text/html")
}
