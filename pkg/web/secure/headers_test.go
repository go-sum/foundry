package secure

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-sum/web"
)

func TestHeaders_SetsAllConfiguredHeaders(t *testing.T) {
	cfg := DefaultHeadersConfig()
	mw := Headers(cfg)

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	resp, _ := handler(web.NewContext(context.Background(), req))

	want := map[string]string{
		"Content-Security-Policy":      cfg.ContentSecurityPolicy,
		"Strict-Transport-Security":    cfg.StrictTransportSecurity,
		"X-Frame-Options":              cfg.XFrameOptions,
		"X-Content-Type-Options":       cfg.XContentTypeOptions,
		"Referrer-Policy":              cfg.ReferrerPolicy,
		"Permissions-Policy":           cfg.PermissionsPolicy,
		"Cross-Origin-Opener-Policy":   cfg.CrossOriginOpenerPolicy,
		"Cross-Origin-Resource-Policy": cfg.CrossOriginResourcePolicy,
	}

	for header, expected := range want {
		got := resp.Headers.Get(header)
		if got != expected {
			t.Errorf("Header %q = %q, want %q", header, got, expected)
		}
	}
}

func TestHeaders_EmptyConfigValuesAreNotSet(t *testing.T) {
	cfg := HeadersConfig{
		XFrameOptions: "DENY",
		// All others are empty.
	}
	mw := Headers(cfg)

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	resp, _ := handler(web.NewContext(context.Background(), req))

	if got := resp.Headers.Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want %q", got, "DENY")
	}

	emptyHeaders := []string{
		"Content-Security-Policy",
		"Strict-Transport-Security",
		"X-Content-Type-Options",
		"Referrer-Policy",
		"Permissions-Policy",
		"Cross-Origin-Opener-Policy",
		"Cross-Origin-Resource-Policy",
	}
	for _, header := range emptyHeaders {
		if resp.Headers.Has(header) {
			t.Errorf("Header %q should not be set when config value is empty", header)
		}
	}
}

func TestDefaultHeadersConfig_AllFieldsNonEmpty(t *testing.T) {
	cfg := DefaultHeadersConfig()

	if cfg.ContentSecurityPolicy == "" {
		t.Error("ContentSecurityPolicy is empty")
	}
	if cfg.StrictTransportSecurity == "" {
		t.Error("StrictTransportSecurity is empty")
	}
	if cfg.XFrameOptions == "" {
		t.Error("XFrameOptions is empty")
	}
	if cfg.XContentTypeOptions == "" {
		t.Error("XContentTypeOptions is empty")
	}
	if cfg.ReferrerPolicy == "" {
		t.Error("ReferrerPolicy is empty")
	}
	if cfg.PermissionsPolicy == "" {
		t.Error("PermissionsPolicy is empty")
	}
	if cfg.CrossOriginOpenerPolicy == "" {
		t.Error("CrossOriginOpenerPolicy is empty")
	}
	if cfg.CrossOriginResourcePolicy == "" {
		t.Error("CrossOriginResourcePolicy is empty")
	}
}

func TestHeaders_SetOnResponse_NotRequest(t *testing.T) {
	cfg := DefaultHeadersConfig()
	mw := Headers(cfg)

	var capturedReq web.Request
	handler := mw(func(c *web.Context) (web.Response, error) {
		capturedReq = c.Request
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	resp, _ := handler(web.NewContext(context.Background(), req))

	// Response should have headers.
	if resp.Headers.Get("X-Frame-Options") != "DENY" {
		t.Error("response missing X-Frame-Options header")
	}

	// Request should NOT have security headers.
	if capturedReq.Headers.Has("X-Frame-Options") {
		t.Error("request should not have X-Frame-Options header")
	}
	if capturedReq.Headers.Has("Content-Security-Policy") {
		t.Error("request should not have Content-Security-Policy header")
	}
}

func TestHeaders_PreservesResponseStatus(t *testing.T) {
	mw := Headers(DefaultHeadersConfig())
	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusCreated), nil
	})

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/"})
	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusCreated {
		t.Errorf("status = %d, want %d", resp.Status, http.StatusCreated)
	}
}
