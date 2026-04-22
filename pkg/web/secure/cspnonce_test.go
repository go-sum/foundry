package secure

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/go-sum/web"
)

func TestCSPNonce_SetsHeaderWithNonce(t *testing.T) {
	mw := CSPNonce(CSPNonceConfig{
		CSPTemplate: "default-src 'self'; script-src 'nonce-{nonce}'",
	})

	var capturedNonce string
	handler := mw(func(c *web.Context) (web.Response, error) {
		capturedNonce = Nonce(c)
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	resp, _ := handler(web.NewContext(context.Background(), req))

	want := "default-src 'self'; script-src 'nonce-" + capturedNonce + "'"
	got := resp.Headers.Get("Content-Security-Policy")
	if got != want {
		t.Errorf("Content-Security-Policy = %q, want %q", got, want)
	}

	if len(capturedNonce) != 22 {
		t.Errorf("nonce length = %d, want 22", len(capturedNonce))
	}
}

func TestCSPNonce_UniquePerRequest(t *testing.T) {
	mw := CSPNonce(CSPNonceConfig{
		CSPTemplate: "default-src 'self'; script-src 'nonce-{nonce}'",
	})

	var nonce1, nonce2 string

	capture1 := mw(func(c *web.Context) (web.Response, error) {
		nonce1 = Nonce(c)
		return web.Respond(http.StatusOK), nil
	})
	capture2 := mw(func(c *web.Context) (web.Response, error) {
		nonce2 = Nonce(c)
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	if _, err := capture1(web.NewContext(context.Background(), req)); err != nil {
		t.Fatalf("capture1: %v", err)
	}
	if _, err := capture2(web.NewContext(context.Background(), req)); err != nil {
		t.Fatalf("capture2: %v", err)
	}

	if nonce1 == nonce2 {
		t.Errorf("expected unique nonces per request, got identical: %q", nonce1)
	}
}

func TestCSPNonce_EmptyTemplate_NoHeader(t *testing.T) {
	mw := CSPNonce(CSPNonceConfig{})

	var capturedNonce string
	handler := mw(func(c *web.Context) (web.Response, error) {
		capturedNonce = Nonce(c)
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Headers.Has("Content-Security-Policy") {
		t.Errorf("Content-Security-Policy header should not be set when CSPTemplate is empty")
	}

	if capturedNonce == "" {
		t.Error("nonce should be generated even when CSPTemplate is empty")
	}
}

func TestNonce_NoValueInContext_ReturnsEmpty(t *testing.T) {
	got := Nonce(nil)
	if got != "" {
		t.Errorf("Nonce(nil) = %q, want %q", got, "")
	}
}

func TestCSPNonce_ReplacesAllOccurrences(t *testing.T) {
	mw := CSPNonce(CSPNonceConfig{
		CSPTemplate: "script-src 'nonce-{nonce}'; style-src 'nonce-{nonce}'",
	})

	var capturedNonce string
	handler := mw(func(c *web.Context) (web.Response, error) {
		capturedNonce = Nonce(c)
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	resp, _ := handler(web.NewContext(context.Background(), req))

	got := resp.Headers.Get("Content-Security-Policy")
	want := "script-src 'nonce-" + capturedNonce + "'; style-src 'nonce-" + capturedNonce + "'"
	if got != want {
		t.Errorf("Content-Security-Policy = %q, want %q", got, want)
	}

	// Verify no "{nonce}" placeholder remains.
	if strings.Contains(got, "{nonce}") {
		t.Error("Content-Security-Policy still contains unreplaced {nonce} placeholder")
	}
}
