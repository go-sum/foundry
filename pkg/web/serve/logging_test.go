package serve

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-sum/web"
)

func TestRequestIDMiddleware_AddsHeader(t *testing.T) {
	handler := func(c *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	}

	chained := web.Chain(handler, web.WithRequestID())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := web.NewContext(req.Context(), adapt_fromRequest(req))
	resp, _ := chained(c)

	id := resp.Headers.Get("X-Request-Id")
	if id == "" {
		t.Fatal("X-Request-Id header is empty")
	}
	if len(id) != 32 {
		t.Errorf("X-Request-Id length = %d, want 32 (16 bytes hex-encoded)", len(id))
	}
	for _, ch := range id {
		if !isHexChar(ch) {
			t.Errorf("X-Request-Id contains non-hex character: %c", ch)
			break
		}
	}
}

func TestRequestIDMiddleware_StoresInContext(t *testing.T) {
	var capturedID string

	handler := func(c *web.Context) (web.Response, error) {
		capturedID = web.RequestID(c)
		return web.Text(http.StatusOK, "ok"), nil
	}

	chained := web.Chain(handler, web.WithRequestID())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := web.NewContext(req.Context(), adapt_fromRequest(req))
	resp, _ := chained(c)

	if capturedID == "" {
		t.Fatal("RequestID in context is empty")
	}
	if capturedID != resp.Headers.Get("X-Request-Id") {
		t.Errorf("context ID %q != header ID %q", capturedID, resp.Headers.Get("X-Request-Id"))
	}
}

func TestRequestID_AbsentReturnsEmpty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := web.NewContext(req.Context(), adapt_fromRequest(req))
	if id := web.RequestID(c); id != "" {
		t.Errorf("RequestID without middleware = %q, want empty", id)
	}
}

func TestAccessLogMiddleware_NoPanic(t *testing.T) {
	handler := func(c *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	}

	chained := web.Chain(handler, web.WithRequestID(), AccessLogMiddleware())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c := web.NewContext(req.Context(), adapt_fromRequest(req))

	// Should not panic.
	resp, _ := chained(c)
	if resp.Status != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.Status, http.StatusOK)
	}
}

func TestAccessLogMiddleware_NilURL(t *testing.T) {
	handler := func(c *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	}

	chained := web.Chain(handler, AccessLogMiddleware())

	// Build a context with a nil URL to exercise graceful handling.
	webReq := web.NewRequest(http.MethodGet, &url.URL{})
	c := web.NewContext(context.Background(), webReq)
	c.Request.URL = nil

	// Should not panic.
	_, _ = chained(c)
}

// adapt_fromRequest is a test helper that converts *http.Request to web.Request
// using the package-local FromHTTPRequest function.
func adapt_fromRequest(r *http.Request) web.Request {
	return FromHTTPRequest(r)
}

func isHexChar(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}
