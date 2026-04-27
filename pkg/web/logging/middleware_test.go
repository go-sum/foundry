package logging

import (
	"bytes"
	"context"
	"log/slog"
	"net/url"
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/web"
)

// buildContext creates a minimal *web.Context for middleware tests.
func buildContext() *web.Context {
	u := &url.URL{Path: "/"}
	req := web.NewRequest("GET", u)
	return web.NewContext(context.Background(), req)
}

// TestMiddleware_PropagatesLoggerWithRequestID verifies that when a request ID
// is present in the context, the injected logger emits a request_id attribute.
func TestMiddleware_PropagatesLoggerWithRequestID(t *testing.T) {
	var buf bytes.Buffer
	base := New(Config{Format: FormatText, Output: &buf})

	c := buildContext()
	c.Set(web.RequestIDKey{}, "test-id-xyz")

	var loggedFromCtx *slog.Logger
	inner := func(c *web.Context) (web.Response, error) {
		loggedFromCtx = FromContext(c.Context())
		loggedFromCtx.Info("from inner handler")
		return web.Response{}, nil
	}

	handler := Middleware(base)(inner)
	_, err := handler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loggedFromCtx == nil {
		t.Fatal("FromContext returned nil inside inner handler")
	}
	if loggedFromCtx == base {
		t.Error("expected enriched logger (not the same pointer as base) inside context")
	}

	out := buf.String()
	if !strings.Contains(out, "request_id=test-id-xyz") {
		t.Errorf("expected request_id=test-id-xyz in output, got: %q", out)
	}
}

// TestMiddleware_NoRequestIDWhenAbsent verifies that when no request ID is set,
// the middleware still injects a logger into the context without a request_id
// attribute in the output.
func TestMiddleware_NoRequestIDWhenAbsent(t *testing.T) {
	var buf bytes.Buffer
	base := New(Config{Format: FormatText, Output: &buf})

	c := buildContext()

	var loggedFromCtx *slog.Logger
	inner := func(c *web.Context) (web.Response, error) {
		loggedFromCtx = FromContext(c.Context())
		loggedFromCtx.Info("no request id here")
		return web.Response{}, nil
	}

	handler := Middleware(base)(inner)
	_, err := handler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loggedFromCtx == nil {
		t.Fatal("FromContext returned nil inside inner handler")
	}

	out := buf.String()
	if strings.Contains(out, "request_id") {
		t.Errorf("expected no request_id attribute when absent, got: %q", out)
	}
	if !strings.Contains(out, "no request id here") {
		t.Errorf("expected log message in output, got: %q", out)
	}
}

// TestMiddleware_CallsNext verifies the inner handler is always reached.
func TestMiddleware_CallsNext(t *testing.T) {
	var buf bytes.Buffer
	base := New(Config{Output: &buf})

	c := buildContext()

	called := false
	inner := func(c *web.Context) (web.Response, error) {
		called = true
		return web.Response{}, nil
	}

	handler := Middleware(base)(inner)
	_, err := handler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("inner handler was not called")
	}
}

// TestMiddleware_ReturnsNextResponse verifies the response from the inner
// handler is passed through unchanged.
func TestMiddleware_ReturnsNextResponse(t *testing.T) {
	var buf bytes.Buffer
	base := New(Config{Output: &buf})

	c := buildContext()

	want := web.Response{Status: 201}
	inner := func(c *web.Context) (web.Response, error) {
		return want, nil
	}

	handler := Middleware(base)(inner)
	got, err := handler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != want.Status {
		t.Errorf("response status = %d, want %d", got.Status, want.Status)
	}
}
