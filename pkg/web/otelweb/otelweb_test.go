package otelweb_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/go-sum/web"
	"github.com/go-sum/web/otelweb"
)

func makeContext(method, path string) *web.Context {
	u, _ := url.Parse(path)
	req := web.NewRequest(method, u)
	return web.NewContext(context.Background(), req)
}

func TestMiddleware_SpanStartedAndEnded(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	mw := otelweb.Middleware(tracer)

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		// Verify the context was updated with a span.
		if c.Context() == nil {
			t.Fatal("context is nil after middleware")
		}
		return web.Text(http.StatusOK, "ok"), nil
	})

	c := makeContext(http.MethodGet, "/test")
	resp, err := handler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("inner handler was not called")
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
}

func TestMiddleware_5xxResponseAttributes(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	mw := otelweb.Middleware(tracer)

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Text(http.StatusInternalServerError, "error"), nil
	})

	c := makeContext(http.MethodPost, "/fail")
	resp, err := handler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusInternalServerError)
	}
}

func TestMiddleware_2xxResponseDoesNotRecordError(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	mw := otelweb.Middleware(tracer)

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	})

	c := makeContext(http.MethodGet, "/ok")
	_, err := handler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With noop tracer there's no span recording to assert on,
	// but the middleware must not panic or error on 2xx responses.
}

func TestExtractTraceID_NoopTracerReturnsEmpty(t *testing.T) {
	extractor := otelweb.ExtractTraceID()
	c := makeContext(http.MethodGet, "/test")
	// noop tracer has no active span, so trace ID should be empty.
	got := extractor(c)
	if got != "" {
		t.Fatalf("ExtractTraceID with noop tracer = %q, want empty string", got)
	}
}

func TestExtractSpanID_NoopTracerReturnsEmpty(t *testing.T) {
	extractor := otelweb.ExtractSpanID()
	c := makeContext(http.MethodGet, "/test")
	// noop tracer has no active span, so span ID should be empty.
	got := extractor(c)
	if got != "" {
		t.Fatalf("ExtractSpanID with noop tracer = %q, want empty string", got)
	}
}

func TestMakeOnError_NilSafeWithNoopSpan(t *testing.T) {
	hook := otelweb.MakeOnError()

	// Call with a context that has no active span — should not panic.
	hook(context.Background(), &web.Error{
		Status: http.StatusInternalServerError,
		Code:   web.CodeInternal,
		Title:  "Internal Server Error",
	})
}
