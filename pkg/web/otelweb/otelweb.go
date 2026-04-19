package otelweb

import (
	"cmp"
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-sum/web"
)

// Middleware creates a server span for every request using tracer, setting
// http.request.method and http.response.status_code attributes. On 5xx
// responses and recovered panics it records the error on the span and sets
// span status to codes.Error.
//
// Install BEFORE web.ErrorBoundary so the span is active when the boundary
// runs. Use the OnError hook from web.BoundaryConfig together with
// Middleware to record errors — see MakeOnError.
func Middleware(tracer trace.Tracer) web.Middleware {
	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			ctx, span := tracer.Start(c.Context(), spanName(c),
				trace.WithSpanKind(trace.SpanKindServer),
			)
			defer span.End()

			c.SetContext(ctx)

			resp, err := next(c)

			status := cmp.Or(resp.Status, http.StatusOK)
			span.SetAttributes(
				attribute.String("http.request.method", c.Method()),
				attribute.Int("http.response.status_code", status),
			)

			return resp, err
		}
	}
}

// MakeOnError returns a BoundaryConfig.OnError hook that records classified
// errors on the active span. Install it alongside Middleware.
func MakeOnError() func(ctx context.Context, e *web.Error) {
	return func(ctx context.Context, e *web.Error) {
		span := trace.SpanFromContext(ctx)
		if !span.IsRecording() {
			return
		}
		if e.Status >= 500 {
			span.SetAttributes(attribute.String("error.type", string(e.Code)))
			var cause error
			if e.Cause != nil {
				cause = e.Cause
			} else {
				cause = fmt.Errorf("%s", e.PublicMessage())
			}
			span.RecordError(cause)
			span.SetStatus(codes.Error, e.PublicMessage())
		}
	}
}

// ExtractTraceID returns a BoundaryConfig.TraceID extractor that reads the
// trace ID from the current span context stored in c.Context().
func ExtractTraceID() func(*web.Context) string {
	return func(c *web.Context) string {
		sc := trace.SpanFromContext(c.Context()).SpanContext()
		if !sc.IsValid() {
			return ""
		}
		return sc.TraceID().String()
	}
}

// ExtractSpanID returns a BoundaryConfig.SpanID extractor that reads the
// span ID from the current span context stored in c.Context().
func ExtractSpanID() func(*web.Context) string {
	return func(c *web.Context) string {
		sc := trace.SpanFromContext(c.Context()).SpanContext()
		if !sc.IsValid() {
			return ""
		}
		return sc.SpanID().String()
	}
}

// spanName returns a concise name for the server span.
func spanName(c *web.Context) string {
	return c.Method() + " " + c.URL().Path
}
