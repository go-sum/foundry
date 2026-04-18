package secure

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-sum/web"
)

// strictOriginCfg is the baseline config used by most tests:
// one trusted origin, strict mode (PermitUnknownOrigin = false).
func strictOriginCfg() OriginGuardConfig {
	return OriginGuardConfig{
		TrustedOrigins: []string{"http://example.com"},
	}
}

// originGuardNext returns a handler that sets called=true when invoked.
func originGuardNext(called *bool) web.Handler {
	return func(c *web.Context) (web.Response, error) {
		*called = true
		return web.Respond(http.StatusOK), nil
	}
}

// assertOriginForbidden verifies that err is a *web.Error with status 403.
func assertOriginForbidden(t *testing.T, err error) {
	t.Helper()
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusForbidden {
		t.Fatalf("error status = %d, want %d", webErr.Status, http.StatusForbidden)
	}
}

func TestOriginGuard_SafeMethods_AlwaysPass(t *testing.T) {
	tests := []struct {
		name   string
		method string
	}{
		{"GET passes", http.MethodGet},
		{"HEAD passes", http.MethodHead},
		{"OPTIONS passes", http.MethodOptions},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			handler := OriginGuard(strictOriginCfg())(originGuardNext(&called))

			req := web.NewRequest(tc.method, &url.URL{Path: "/"})
			resp, err := handler(web.NewContext(context.Background(), req))

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Status != http.StatusOK {
				t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
			}
			if !called {
				t.Error("next handler was not called for safe method")
			}
		})
	}
}

func TestOriginGuard_SecFetchSite_Pass(t *testing.T) {
	tests := []struct {
		name          string
		secFetchSite  string
	}{
		{"same-origin passes", "same-origin"},
		{"none passes", "none"},
		{"same-site passes", "same-site"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			handler := OriginGuard(strictOriginCfg())(originGuardNext(&called))

			req := web.NewRequest(http.MethodPost, &url.URL{Path: "/"})
			req.Headers.Set("Sec-Fetch-Site", tc.secFetchSite)

			resp, err := handler(web.NewContext(context.Background(), req))

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Status != http.StatusOK {
				t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
			}
			if !called {
				t.Errorf("next handler was not called for Sec-Fetch-Site: %s", tc.secFetchSite)
			}
		})
	}
}

func TestOriginGuard_OriginHeader_MatchingTrusted_Passes(t *testing.T) {
	called := false
	handler := OriginGuard(strictOriginCfg())(originGuardNext(&called))

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "http://example.com")

	resp, err := handler(web.NewContext(context.Background(), req))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler was not called for matching Origin")
	}
}

func TestOriginGuard_OriginHeader_NonMatchingTrusted_Returns403(t *testing.T) {
	called := false
	handler := OriginGuard(strictOriginCfg())(originGuardNext(&called))

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "http://evil.com")

	_, err := handler(web.NewContext(context.Background(), req))

	assertOriginForbidden(t, err)
	if called {
		t.Error("next handler was called for non-matching Origin")
	}
}

func TestOriginGuard_RefererHeader_MatchingTrusted_Passes(t *testing.T) {
	called := false
	handler := OriginGuard(strictOriginCfg())(originGuardNext(&called))

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/"})
	req.Headers.Set("Referer", "http://example.com/page")

	resp, err := handler(web.NewContext(context.Background(), req))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler was not called for matching Referer")
	}
}

func TestOriginGuard_RefererHeader_ExactMatch_Passes(t *testing.T) {
	called := false
	handler := OriginGuard(strictOriginCfg())(originGuardNext(&called))

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/"})
	req.Headers.Set("Referer", "http://example.com")

	resp, err := handler(web.NewContext(context.Background(), req))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler was not called for exact-matching Referer")
	}
}

func TestOriginGuard_NoOriginInfo_Strict_Returns403(t *testing.T) {
	called := false
	handler := OriginGuard(strictOriginCfg())(originGuardNext(&called))

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/"})
	// No Origin, no Referer, no Sec-Fetch-Site.

	_, err := handler(web.NewContext(context.Background(), req))

	assertOriginForbidden(t, err)
	if called {
		t.Error("next handler was called with no origin info in strict mode")
	}
}

func TestOriginGuard_EmptyTrustedOrigins_OriginHeader_Returns403(t *testing.T) {
	called := false
	cfg := OriginGuardConfig{
		TrustedOrigins: []string{},
	}
	handler := OriginGuard(cfg)(originGuardNext(&called))

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "http://example.com")

	_, err := handler(web.NewContext(context.Background(), req))

	assertOriginForbidden(t, err)
	if called {
		t.Error("next handler was called with empty TrustedOrigins and non-same-origin Origin")
	}
}

func TestOriginGuard_PermitUnknownOrigin_True_NoOriginInfo_Passes(t *testing.T) {
	called := false
	cfg := OriginGuardConfig{
		TrustedOrigins:      []string{"http://example.com"},
		PermitUnknownOrigin: true,
	}
	handler := OriginGuard(cfg)(originGuardNext(&called))

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/"})
	// No Origin, no Referer, no Sec-Fetch-Site.

	resp, err := handler(web.NewContext(context.Background(), req))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler was not called with PermitUnknownOrigin=true and no origin info")
	}
}

func TestOriginGuard_PermitUnknownOrigin_False_NoOriginInfo_Returns403(t *testing.T) {
	called := false
	cfg := OriginGuardConfig{
		TrustedOrigins:      []string{"http://example.com"},
		PermitUnknownOrigin: false,
	}
	handler := OriginGuard(cfg)(originGuardNext(&called))

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/"})
	// No Origin, no Referer, no Sec-Fetch-Site.

	_, err := handler(web.NewContext(context.Background(), req))

	assertOriginForbidden(t, err)
	if called {
		t.Error("next handler was called with PermitUnknownOrigin=false and no origin info")
	}
}

func TestOriginGuard_SecFetchSiteCrossSite_TrustedOrigin_Passes(t *testing.T) {
	called := false
	handler := OriginGuard(strictOriginCfg())(originGuardNext(&called))

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/"})
	req.Headers.Set("Sec-Fetch-Site", "cross-site")
	req.Headers.Set("Origin", "http://example.com")

	resp, err := handler(web.NewContext(context.Background(), req))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler was not called for cross-site with trusted Origin")
	}
}

func TestOriginGuard_SecFetchSiteCrossSite_UntrustedOrigin_Returns403(t *testing.T) {
	called := false
	handler := OriginGuard(strictOriginCfg())(originGuardNext(&called))

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/"})
	req.Headers.Set("Sec-Fetch-Site", "cross-site")
	req.Headers.Set("Origin", "http://evil.com")

	_, err := handler(web.NewContext(context.Background(), req))

	assertOriginForbidden(t, err)
	if called {
		t.Error("next handler was called for cross-site with untrusted Origin")
	}
}

func TestOriginGuard_TrustedOriginFunc_ReturnsTrue_Passes(t *testing.T) {
	called := false
	cfg := OriginGuardConfig{
		TrustedOriginFunc: func(origin string) bool {
			return origin == "http://dynamic.example.com"
		},
	}
	handler := OriginGuard(cfg)(originGuardNext(&called))

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "http://dynamic.example.com")

	resp, err := handler(web.NewContext(context.Background(), req))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler was not called when TrustedOriginFunc returned true")
	}
}

func TestOriginGuard_TrustedOriginFunc_ReturnsFalse_Returns403(t *testing.T) {
	called := false
	cfg := OriginGuardConfig{
		TrustedOriginFunc: func(origin string) bool {
			return false
		},
	}
	handler := OriginGuard(cfg)(originGuardNext(&called))

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/"})
	req.Headers.Set("Origin", "http://example.com")

	_, err := handler(web.NewContext(context.Background(), req))

	assertOriginForbidden(t, err)
	if called {
		t.Error("next handler was called when TrustedOriginFunc returned false")
	}
}

func TestOriginGuard_Skipper_ReturnsTrue_BypassesValidation(t *testing.T) {
	called := false
	cfg := OriginGuardConfig{
		TrustedOrigins: []string{"http://example.com"},
		Skipper: func(_ *web.Context) bool {
			return true
		},
	}
	handler := OriginGuard(cfg)(originGuardNext(&called))

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/"})
	// No origin info at all — would be rejected without skipper.

	resp, err := handler(web.NewContext(context.Background(), req))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler was not called when Skipper returned true")
	}
}

// TestOriginGuard_AllCases is the comprehensive table-driven test covering the
// full decision matrix of the OriginGuard middleware.
func TestOriginGuard_AllCases(t *testing.T) {
	tests := []struct {
		name        string
		cfg         OriginGuardConfig
		method      string
		secFetch    string
		origin      string
		referer     string
		wantCalled  bool
		wantBlocked bool
	}{
		// --- safe methods ---
		{
			name:       "GET no headers passes",
			cfg:        strictOriginCfg(),
			method:     http.MethodGet,
			wantCalled: true,
		},
		{
			name:       "HEAD no headers passes",
			cfg:        strictOriginCfg(),
			method:     http.MethodHead,
			wantCalled: true,
		},
		{
			name:       "OPTIONS no headers passes",
			cfg:        strictOriginCfg(),
			method:     http.MethodOptions,
			wantCalled: true,
		},
		// --- Sec-Fetch-Site fast paths ---
		{
			name:       "POST Sec-Fetch-Site same-origin passes",
			cfg:        strictOriginCfg(),
			method:     http.MethodPost,
			secFetch:   "same-origin",
			wantCalled: true,
		},
		{
			name:       "POST Sec-Fetch-Site none passes",
			cfg:        strictOriginCfg(),
			method:     http.MethodPost,
			secFetch:   "none",
			wantCalled: true,
		},
		{
			name:       "POST Sec-Fetch-Site same-site passes",
			cfg:        strictOriginCfg(),
			method:     http.MethodPost,
			secFetch:   "same-site",
			wantCalled: true,
		},
		// --- cross-site Sec-Fetch-Site ---
		{
			name:       "POST Sec-Fetch-Site cross-site trusted Origin passes",
			cfg:        strictOriginCfg(),
			method:     http.MethodPost,
			secFetch:   "cross-site",
			origin:     "http://example.com",
			wantCalled: true,
		},
		{
			name:        "POST Sec-Fetch-Site cross-site untrusted Origin blocked",
			cfg:         strictOriginCfg(),
			method:      http.MethodPost,
			secFetch:    "cross-site",
			origin:      "http://evil.com",
			wantCalled:  false,
			wantBlocked: true,
		},
		// --- Origin fallback ---
		{
			name:       "POST matching Origin passes",
			cfg:        strictOriginCfg(),
			method:     http.MethodPost,
			origin:     "http://example.com",
			wantCalled: true,
		},
		{
			name:        "POST non-matching Origin blocked",
			cfg:         strictOriginCfg(),
			method:      http.MethodPost,
			origin:      "http://attacker.com",
			wantCalled:  false,
			wantBlocked: true,
		},
		// --- Referer fallback ---
		{
			name:       "POST matching Referer with path passes",
			cfg:        strictOriginCfg(),
			method:     http.MethodPost,
			referer:    "http://example.com/page",
			wantCalled: true,
		},
		// --- no origin info ---
		{
			name:        "POST no origin info strict mode blocked",
			cfg:         strictOriginCfg(),
			method:      http.MethodPost,
			wantCalled:  false,
			wantBlocked: true,
		},
		{
			name: "POST no origin info PermitUnknownOrigin true passes",
			cfg: OriginGuardConfig{
				TrustedOrigins:      []string{"http://example.com"},
				PermitUnknownOrigin: true,
			},
			method:     http.MethodPost,
			wantCalled: true,
		},
		// --- empty TrustedOrigins ---
		{
			name:        "POST empty TrustedOrigins with Origin header blocked",
			cfg:         OriginGuardConfig{TrustedOrigins: []string{}},
			method:      http.MethodPost,
			origin:      "http://example.com",
			wantCalled:  false,
			wantBlocked: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			handler := OriginGuard(tc.cfg)(originGuardNext(&called))

			req := web.NewRequest(tc.method, &url.URL{Path: "/"})
			if tc.secFetch != "" {
				req.Headers.Set("Sec-Fetch-Site", tc.secFetch)
			}
			if tc.origin != "" {
				req.Headers.Set("Origin", tc.origin)
			}
			if tc.referer != "" {
				req.Headers.Set("Referer", tc.referer)
			}

			resp, err := handler(web.NewContext(context.Background(), req))

			if tc.wantBlocked {
				assertOriginForbidden(t, err)
				if called {
					t.Error("next handler was called but should have been blocked")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if resp.Status != http.StatusOK {
					t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
				}
				if !called {
					t.Error("next handler was not called but should have passed")
				}
			}
		})
	}
}
