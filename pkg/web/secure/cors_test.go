package secure

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-sum/foundry/pkg/web"
)

func TestCORS_NoOriginHeader_PassesThroughWithoutCORSHeaders(t *testing.T) {
	mw := CORS(CORSConfig{
		AllowOrigins: []string{"https://example.com"},
	})

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler not called")
	}
	if resp.Headers.Has("Access-Control-Allow-Origin") {
		t.Error("Access-Control-Allow-Origin should not be set without Origin header")
	}
}

func TestCORS_AllowedOrigin_SetsHeader(t *testing.T) {
	mw := CORS(CORSConfig{
		AllowOrigins: []string{"https://example.com"},
	})

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "https://example.com")

	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	got := resp.Headers.Get("Access-Control-Allow-Origin")
	if got != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "https://example.com")
	}
	if resp.Headers.Get("Vary") != "Origin" {
		t.Errorf("Vary = %q, want %q", resp.Headers.Get("Vary"), "Origin")
	}
}

func TestCORS_DisallowedOrigin_PassesThroughWithoutCORSHeaders(t *testing.T) {
	mw := CORS(CORSConfig{
		AllowOrigins: []string{"https://example.com"},
	})

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "https://evil.com")

	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler not called for disallowed origin")
	}
	if resp.Headers.Has("Access-Control-Allow-Origin") {
		t.Error("Access-Control-Allow-Origin should not be set for disallowed origin")
	}
}

func TestCORS_Wildcard_SetsWildcardOrigin(t *testing.T) {
	mw := CORS(CORSConfig{
		AllowOrigins: []string{"*"},
	})

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "https://anything.com")

	resp, _ := handler(web.NewContext(context.Background(), req))

	got := resp.Headers.Get("Access-Control-Allow-Origin")
	if got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
	// Wildcard should NOT set Vary: Origin.
	if resp.Headers.Has("Vary") {
		t.Error("Vary should not be set for wildcard origin")
	}
}

func TestCORS_PreflightOPTIONS_Returns204WithCORSHeaders(t *testing.T) {
	mw := CORS(CORSConfig{
		AllowOrigins: []string{"https://example.com"},
		AllowHeaders: []string{"Content-Type", "X-CSRF-Token"},
	})

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodOptions, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "https://example.com")
	req.Headers.Set("Access-Control-Request-Method", http.MethodPost)

	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusNoContent)
	}
	if called {
		t.Error("next handler should not be called for preflight")
	}

	got := resp.Headers.Get("Access-Control-Allow-Origin")
	if got != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "https://example.com")
	}

	methods := resp.Headers.Get("Access-Control-Allow-Methods")
	if methods == "" {
		t.Error("Access-Control-Allow-Methods should be set on preflight")
	}

	headers := resp.Headers.Get("Access-Control-Allow-Headers")
	if headers != "Content-Type, X-CSRF-Token" {
		t.Errorf("Access-Control-Allow-Headers = %q, want %q", headers, "Content-Type, X-CSRF-Token")
	}
}

func TestCORS_AllowCredentials_SetsHeader(t *testing.T) {
	mw := CORS(CORSConfig{
		AllowOrigins:     []string{"https://example.com"},
		AllowCredentials: true,
	})

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "https://example.com")

	resp, _ := handler(web.NewContext(context.Background(), req))

	got := resp.Headers.Get("Access-Control-Allow-Credentials")
	if got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want %q", got, "true")
	}
}

func TestCORS_CaseInsensitiveOriginMatching(t *testing.T) {
	mw := CORS(CORSConfig{
		AllowOrigins: []string{"https://Example.COM"},
	})

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "https://example.com")

	resp, _ := handler(web.NewContext(context.Background(), req))

	got := resp.Headers.Get("Access-Control-Allow-Origin")
	if got != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q (case-insensitive match)", got, "https://example.com")
	}
}

func TestCORS_Skipper_BypassesCORS(t *testing.T) {
	mw := CORS(CORSConfig{
		AllowOrigins: []string{"https://example.com"},
		Skipper: func(_ *web.Context) bool {
			return true
		},
	})

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "https://example.com")

	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler not called when skipper is true")
	}
	// When skipped, no CORS headers should be set.
	if resp.Headers.Has("Access-Control-Allow-Origin") {
		t.Error("CORS headers should not be set when skipper returns true")
	}
}

func TestCORS_ExposeHeaders(t *testing.T) {
	mw := CORS(CORSConfig{
		AllowOrigins:  []string{"https://example.com"},
		ExposeHeaders: []string{"X-Request-ID", "X-Custom"},
	})

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "https://example.com")

	resp, _ := handler(web.NewContext(context.Background(), req))

	got := resp.Headers.Get("Access-Control-Expose-Headers")
	if got != "X-Request-ID, X-Custom" {
		t.Errorf("Access-Control-Expose-Headers = %q, want %q", got, "X-Request-ID, X-Custom")
	}
}

func TestCORS_PreservesExistingVary(t *testing.T) {
	mw := CORS(CORSConfig{
		AllowOrigins: []string{"https://example.com"},
	})

	handler := mw(func(c *web.Context) (web.Response, error) {
		resp := web.Respond(http.StatusOK)
		resp.Headers.Set("Vary", "Accept-Encoding")
		return resp, nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "https://example.com")

	resp, _ := handler(web.NewContext(context.Background(), req))
	if got := resp.Headers.Get("Vary"); got != "Origin, Accept-Encoding" {
		t.Errorf("Vary = %q, want %q", got, "Origin, Accept-Encoding")
	}
}

func TestCORS_Preflight_EchoesRequestedHeadersWhenNotConfigured(t *testing.T) {
	mw := CORS(CORSConfig{
		AllowOrigins: []string{"https://example.com"},
	})

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodOptions, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "https://example.com")
	req.Headers.Set("Access-Control-Request-Method", http.MethodPost)
	req.Headers.Set("Access-Control-Request-Headers", "X-CSRF-Token, Content-Type")

	resp, _ := handler(web.NewContext(context.Background(), req))
	if got := resp.Headers.Get("Access-Control-Allow-Headers"); got != "X-CSRF-Token, Content-Type" {
		t.Errorf("Access-Control-Allow-Headers = %q, want %q", got, "X-CSRF-Token, Content-Type")
	}
	if got := resp.Headers.Get("Vary"); got != "Origin, Access-Control-Request-Method, Access-Control-Request-Headers" {
		t.Errorf("Vary = %q, want %q", got, "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")
	}
}

func TestCORS_OptionsWithOriginButNoAccessControlRequestMethod_PassesThrough(t *testing.T) {
	mw := CORS(CORSConfig{
		AllowOrigins: []string{"https://example.com"},
	})

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodOptions, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "https://example.com")

	resp, _ := handler(web.NewContext(context.Background(), req))
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Fatal("next handler not called")
	}
	if got := resp.Headers.Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "https://example.com")
	}
	if got := resp.Headers.Get("Access-Control-Allow-Methods"); got != "" {
		t.Fatalf("Access-Control-Allow-Methods = %q, want empty", got)
	}
}

func TestCORS_MaxAge(t *testing.T) {
	mw := CORS(CORSConfig{
		AllowOrigins: []string{"https://example.com"},
		MaxAge:       3600,
	})

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "https://example.com")

	resp, _ := handler(web.NewContext(context.Background(), req))

	got := resp.Headers.Get("Access-Control-Max-Age")
	if got != "3600" {
		t.Errorf("Access-Control-Max-Age = %q, want %q", got, "3600")
	}
}

func TestCORS_MaxAge_NoTrailingZeros(t *testing.T) {
	mw := CORS(CORSConfig{
		AllowOrigins: []string{"https://example.com"},
		MaxAge:       86399,
	})

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "https://example.com")

	resp, _ := handler(web.NewContext(context.Background(), req))

	got := resp.Headers.Get("Access-Control-Max-Age")
	if got != "86399" {
		t.Errorf("Access-Control-Max-Age = %q, want %q", got, "86399")
	}
}

func TestCORS_WildcardWithCredentials_SetsOriginNotWildcard(t *testing.T) {
	mw := CORS(CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	})

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "https://example.com")

	resp, _ := handler(web.NewContext(context.Background(), req))

	// When AllowCredentials is true, wildcard should be replaced with actual origin.
	got := resp.Headers.Get("Access-Control-Allow-Origin")
	if got != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q (wildcard+credentials should use actual origin)", got, "https://example.com")
	}
	if resp.Headers.Get("Vary") != "Origin" {
		t.Errorf("Vary = %q, want %q (should Vary when using actual origin)", resp.Headers.Get("Vary"), "Origin")
	}
}

func TestCORS_DefaultAllowMethods(t *testing.T) {
	mw := CORS(CORSConfig{
		AllowOrigins: []string{"https://example.com"},
		// AllowMethods not set -- should use defaults.
	})

	handler := mw(func(c *web.Context) (web.Response, error) {
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodOptions, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "https://example.com")
	req.Headers.Set("Access-Control-Request-Method", http.MethodGet)

	resp, _ := handler(web.NewContext(context.Background(), req))

	methods := resp.Headers.Get("Access-Control-Allow-Methods")
	if methods == "" {
		t.Fatal("Access-Control-Allow-Methods should be set on preflight with defaults")
	}
	// Default methods should include all common methods.
	want := "GET, HEAD, PUT, PATCH, POST, DELETE"
	if methods != want {
		t.Errorf("Access-Control-Allow-Methods = %q, want %q", methods, want)
	}
}
