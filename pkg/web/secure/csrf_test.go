package secure

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/cookiecodec"
	websession "github.com/go-sum/foundry/pkg/web/session"
)

func testCSRFKey() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

func testCSRFConfig() CSRFConfig {
	cfg := DefaultCSRFConfig()
	cfg.Key = testCSRFKey()
	cfg.TokenTTL = time.Hour
	cfg.AllowMissingOrigin = true // token-focused tests; origin tests use explicit configs
	return cfg
}

// assertForbidden asserts that err is a *web.Error with status 403 and the given message.
func assertForbidden(t *testing.T, err error, wantMsg string) {
	t.Helper()
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusForbidden {
		t.Fatalf("error status = %d, want %d", webErr.Status, http.StatusForbidden)
	}
	if webErr.Message != wantMsg {
		t.Errorf("error message = %q, want %q", webErr.Message, wantMsg)
	}
}

func TestCSRF_GET_StoresTokenInContext(t *testing.T) {
	cfg := testCSRFConfig()
	mw := CSRF(cfg)

	var capturedCtx *web.Context
	handler := mw(func(c *web.Context) (web.Response, error) {
		capturedCtx = c
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/test"})
	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}

	tok := CSRFToken(capturedCtx)
	if tok == "" {
		t.Fatal("CSRFToken() returned empty string after GET request")
	}

	// The token should be verifiable.
	if err := VerifyToken(cfg.Key, "csrf", tok); err != nil {
		t.Errorf("issued token failed verification: %v", err)
	}

	// Token should also be set as a cookie on the response.
	cookieHeader := resp.Headers.Get("Set-Cookie")
	if !strings.HasPrefix(cookieHeader, "csrf=") {
		t.Errorf("Set-Cookie = %q, want prefix %q", cookieHeader, "csrf=")
	}
}

func TestSameOriginBase_UsesNormalizedRequestURLScheme(t *testing.T) {
	req := web.NewRequest(http.MethodPost, &url.URL{Scheme: "https", Host: "example.com", Path: "/submit"})

	got := sameOriginBase(web.NewContext(context.Background(), req))
	if got != "https://example.com" {
		t.Fatalf("sameOriginBase() = %q, want %q", got, "https://example.com")
	}
}

func TestSameOriginBase_IgnoresRawXForwardedProto(t *testing.T) {
	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/submit"})
	req.SetHost("example.com")
	req.Headers.Set("X-Forwarded-Proto", "https")

	got := sameOriginBase(web.NewContext(context.Background(), req))
	if got != "http://example.com" {
		t.Fatalf("sameOriginBase() = %q, want %q", got, "http://example.com")
	}
}

func TestCSRF_POST_MissingToken_Returns403(t *testing.T) {
	mw := CSRF(testCSRFConfig())
	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/test"})
	_, err := handler(web.NewContext(context.Background(), req))

	assertForbidden(t, err, "CSRF token missing")
	if called {
		t.Error("next handler was called despite missing CSRF token")
	}
}

func TestCSRF_POST_ValidTokenInHeader_PassesThrough(t *testing.T) {
	cfg := testCSRFConfig()
	mw := CSRF(cfg)

	tok, err := IssueToken(cfg.Key, "csrf", cfg.TokenTTL)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/test"})
	req.Headers.Set("X-CSRF-Token", tok)
	req.Headers.Set("Cookie", "csrf="+tok)

	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler was not called despite valid CSRF token")
	}
}

// TestCSRF_POST_ValidTokenInFormBody_PassesThrough verifies that a token submitted
// in the form body (application/x-www-form-urlencoded) is accepted. The middleware
// uses Clone so the downstream handler can still read the full original body.
func TestCSRF_POST_ValidTokenInFormBody_PassesThrough(t *testing.T) {
	cfg := testCSRFConfig()
	mw := CSRF(cfg)

	tok, err := IssueToken(cfg.Key, "csrf", cfg.TokenTTL)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	var handlerBodyData string
	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		// Body must still be fully readable by the handler after CSRF peeked it.
		fd, fdErr := c.Request.FormData()
		if fdErr != nil {
			t.Errorf("handler FormData() error = %v", fdErr)
		} else {
			handlerBodyData = fd.Values.Get("name")
		}
		return web.Respond(http.StatusOK), nil
	})

	body := "_csrf=" + tok + "&name=testvalue"
	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/test"})
	req.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Headers.Set("Cookie", "csrf="+tok)
	req.SetBody(io.NopCloser(strings.NewReader(body)))

	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200; form body CSRF token must be accepted", resp.Status)
	}
	if !called {
		t.Error("next handler was not called")
	}
	if handlerBodyData != "testvalue" {
		t.Errorf("handler body field = %q, want %q; body must survive CSRF peek", handlerBodyData, "testvalue")
	}
}

// TestCSRF_POST_FormBodyMissingToken_Returns403 verifies that an urlencoded POST
// with no token anywhere (not header, query, or body) still returns 403.
func TestCSRF_POST_FormBodyMissingToken_Returns403(t *testing.T) {
	cfg := testCSRFConfig()
	mw := CSRF(cfg)

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/test"})
	req.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBody(io.NopCloser(strings.NewReader("name=testvalue"))) // no _csrf field

	_, err := handler(web.NewContext(context.Background(), req))

	assertForbidden(t, err, "CSRF token missing")
	if called {
		t.Error("next handler was called despite missing CSRF token")
	}
}

// TestCSRF_POST_JSONBody_NotConsumedByFormPeek verifies that a JSON POST body
// is not consumed or corrupted by the form-peek code path. The content-type
// guard must prevent Clone/FormData from being called for non-form requests.
func TestCSRF_POST_JSONBody_NotConsumedByFormPeek(t *testing.T) {
	cfg := testCSRFConfig()
	mw := CSRF(cfg)

	tok, err := IssueToken(cfg.Key, "csrf", cfg.TokenTTL)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	var bodyData string
	handler := mw(func(c *web.Context) (web.Response, error) {
		b, _ := c.Request.Bytes()
		bodyData = string(b)
		return web.Respond(http.StatusOK), nil
	})

	const jsonBody = `{"key":"value"}`
	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/test"})
	req.Headers.Set("Content-Type", "application/json")
	req.Headers.Set("X-CSRF-Token", tok)
	req.Headers.Set("Cookie", "csrf="+tok)
	req.SetBody(io.NopCloser(strings.NewReader(jsonBody)))

	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.Status)
	}
	if bodyData != jsonBody {
		t.Errorf("handler body = %q, want %q; JSON body must not be consumed by CSRF middleware", bodyData, jsonBody)
	}
}

func TestCSRF_POST_InvalidToken_Returns403(t *testing.T) {
	mw := CSRF(testCSRFConfig())
	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/test"})
	req.Headers.Set("X-CSRF-Token", "totally-invalid-token")

	_, err := handler(web.NewContext(context.Background(), req))

	assertForbidden(t, err, "CSRF token invalid")
	if called {
		t.Error("next handler was called despite invalid CSRF token")
	}
}

func TestCSRF_POST_ExpiredToken_Returns403(t *testing.T) {
	cfg := testCSRFConfig()
	mw := CSRF(cfg)

	tok, err := IssueToken(cfg.Key, "csrf", time.Nanosecond)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	time.Sleep(time.Millisecond) // 1ns TTL is always expired after 1ms

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/test"})
	req.Headers.Set("X-CSRF-Token", tok)

	_, err = handler(web.NewContext(context.Background(), req))

	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusForbidden {
		t.Fatalf("error status = %d, want %d", webErr.Status, http.StatusForbidden)
	}
	if called {
		t.Error("next handler was called despite expired CSRF token")
	}
}

func TestCSRF_Skipper_BypassesValidation(t *testing.T) {
	cfg := testCSRFConfig()
	cfg.Skipper = func(c *web.Context) bool {
		return true
	}
	mw := CSRF(cfg)

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	// POST with no token should pass when skipper returns true.
	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/test"})
	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler was not called when skipper returned true")
	}
}

func TestCSRFToken_NoTokenInContext_ReturnsEmpty(t *testing.T) {
	tok := CSRFToken(nil)
	if tok != "" {
		t.Errorf("CSRFToken() = %q, want empty string", tok)
	}
}

func TestCSRF_PanicsOnShortKey(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for short key, got none")
		}
	}()
	CSRF(CSRFConfig{Key: make([]byte, 16)})
}

func TestCSRF_UnsafeMethods(t *testing.T) {
	// All unsafe methods should require CSRF token.
	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	mw := CSRF(testCSRFConfig())

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			handler := mw(func(c *web.Context) (web.Response, error) {
				return web.Respond(http.StatusOK), nil
			})

			req := web.NewRequest(method, &url.URL{Path: "/test"})
			_, err := handler(web.NewContext(context.Background(), req))

			var webErr *web.Error
			if !errors.As(err, &webErr) {
				t.Fatalf("expected *web.Error for %s without token, got %T: %v", method, err, err)
			}
			if webErr.Status != http.StatusForbidden {
				t.Errorf("status = %d, want %d for %s without token", webErr.Status, http.StatusForbidden, method)
			}
		})
	}
}

func TestCSRF_SafeMethods_NoTokenRequired(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	mw := CSRF(testCSRFConfig())

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			called := false
			handler := mw(func(c *web.Context) (web.Response, error) {
				called = true
				return web.Respond(http.StatusOK), nil
			})

			req := web.NewRequest(method, &url.URL{Path: "/test"})
			resp, _ := handler(web.NewContext(context.Background(), req))

			if resp.Status != http.StatusOK {
				t.Errorf("status = %d, want %d for safe method %s", resp.Status, http.StatusOK, method)
			}
			if !called {
				t.Errorf("next handler not called for safe method %s", method)
			}
		})
	}
}

func TestCSRF_POST_MissingCookie_Returns403(t *testing.T) {
	cfg := testCSRFConfig()
	mw := CSRF(cfg)

	tok, err := IssueToken(cfg.Key, "csrf", cfg.TokenTTL)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/test"})
	req.Headers.Set("X-CSRF-Token", tok)
	// No Cookie header set — double-submit must fail.

	_, err = handler(web.NewContext(context.Background(), req))

	assertForbidden(t, err, "CSRF token mismatch")
	if called {
		t.Error("next handler was called despite missing CSRF cookie")
	}
}

func TestCSRF_POST_CookieMismatch_Returns403(t *testing.T) {
	cfg := testCSRFConfig()
	mw := CSRF(cfg)

	tok, err := IssueToken(cfg.Key, "csrf", cfg.TokenTTL)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/test"})
	req.Headers.Set("X-CSRF-Token", tok)
	req.Headers.Set("Cookie", "csrf=wrong-cookie-value") // cookie doesn't match token

	_, err = handler(web.NewContext(context.Background(), req))

	assertForbidden(t, err, "CSRF token mismatch")
	if called {
		t.Error("next handler was called despite cookie mismatch")
	}
}

func TestCSRF_POST_ReRenderedForm_TokenInContext(t *testing.T) {
	cfg := testCSRFConfig()
	mw := CSRF(cfg)

	tok, err := IssueToken(cfg.Key, "csrf", cfg.TokenTTL)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	var contextToken string
	handler := mw(func(c *web.Context) (web.Response, error) {
		contextToken = CSRFToken(c)
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/test"})
	req.Headers.Set("X-CSRF-Token", tok)
	req.Headers.Set("Cookie", "csrf="+tok)

	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if contextToken == "" {
		t.Error("CSRFToken(ctx) was empty in handler after successful POST verification")
	}
	// The fresh token in context should differ from the submitted token.
	if contextToken == tok {
		t.Error("CSRFToken(ctx) is the same as the submitted token; expected a freshly issued token")
	}
}

func TestCSRF_Response_SetsCookieWithToken(t *testing.T) {
	cfg := testCSRFConfig()
	mw := CSRF(cfg)

	var capturedToken string
	handler := mw(func(c *web.Context) (web.Response, error) {
		capturedToken = CSRFToken(c)
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/test"})
	resp, _ := handler(web.NewContext(context.Background(), req))

	if capturedToken == "" {
		t.Fatal("CSRFToken empty in context")
	}

	cookieHeader := resp.Headers.Get("Set-Cookie")
	wantPrefix := "csrf=" + capturedToken
	if !strings.HasPrefix(cookieHeader, wantPrefix) {
		t.Errorf("Set-Cookie = %q, want prefix %q", cookieHeader, wantPrefix)
	}
	if !strings.Contains(cookieHeader, "SameSite=Lax") {
		t.Errorf("Set-Cookie missing SameSite=Lax: %q", cookieHeader)
	}
	if !strings.Contains(cookieHeader, "Path=/") {
		t.Errorf("Set-Cookie missing Path=/: %q", cookieHeader)
	}
}

func TestCSRF_CookieSameSite(t *testing.T) {
	tests := []struct {
		name             string
		cookieSameSite   string
		wantSameSite     string
		wantSecureForced bool
	}{
		{
			name:           "default (empty) uses Lax",
			cookieSameSite: "",
			wantSameSite:   "Lax",
		},
		{
			name:           "Lax is preserved",
			cookieSameSite: "Lax",
			wantSameSite:   "Lax",
		},
		{
			name:           "Strict is preserved",
			cookieSameSite: "Strict",
			wantSameSite:   "Strict",
		},
		{
			name:             "None is preserved and forces Secure",
			cookieSameSite:   "None",
			wantSameSite:     "None",
			wantSecureForced: true,
		},
		{
			name:           "invalid value defaults to Lax",
			cookieSameSite: "invalid",
			wantSameSite:   "Lax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testCSRFConfig()
			cfg.CookieSameSite = tt.cookieSameSite
			mw := CSRF(cfg)

			handler := mw(func(c *web.Context) (web.Response, error) {
				return web.Respond(http.StatusOK), nil
			})

			req := web.NewRequest(http.MethodGet, &url.URL{Path: "/test"})
			resp, _ := handler(web.NewContext(context.Background(), req))

			if resp.Status != http.StatusOK {
				t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
			}

			cookieHeader := resp.Headers.Get("Set-Cookie")
			wantSameSiteAttr := "SameSite=" + tt.wantSameSite
			if !strings.Contains(cookieHeader, wantSameSiteAttr) {
				t.Errorf("Set-Cookie = %q, want to contain %q", cookieHeader, wantSameSiteAttr)
			}

			if tt.wantSecureForced {
				if !strings.Contains(cookieHeader, "Secure") {
					t.Errorf("Set-Cookie = %q, want Secure attribute when SameSite=None", cookieHeader)
				}
			}
		})
	}
}

func TestCSRF_SecFetchSite_SameOrigin_Bypasses_OriginCheck(t *testing.T) {
	cfg := testCSRFConfig()
	mw := CSRF(cfg)

	tok, _ := IssueToken(cfg.Key, "csrf", cfg.TokenTTL)
	handler := mw(func(c *web.Context) (web.Response, error) { return web.Respond(http.StatusOK), nil })

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/test"})
	req.Headers.Set("X-CSRF-Token", tok)
	req.Headers.Set("Cookie", "csrf="+tok)
	req.Headers.Set("Sec-Fetch-Site", "same-origin")

	resp, _ := handler(web.NewContext(context.Background(), req))
	if resp.Status != http.StatusOK {
		t.Fatalf("same-origin Sec-Fetch-Site: status = %d, want 200", resp.Status)
	}
}

func TestCSRF_SecFetchSite_CrossSite_Blocks(t *testing.T) {
	cfg := testCSRFConfig()
	mw := CSRF(cfg)

	tok, _ := IssueToken(cfg.Key, "csrf", cfg.TokenTTL)
	handler := mw(func(c *web.Context) (web.Response, error) { return web.Respond(http.StatusOK), nil })

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/test"})
	req.Headers.Set("X-CSRF-Token", tok)
	req.Headers.Set("Cookie", "csrf="+tok)
	req.Headers.Set("Sec-Fetch-Site", "cross-site")

	_, err := handler(web.NewContext(context.Background(), req))

	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusForbidden {
		t.Fatalf("cross-site Sec-Fetch-Site: status = %d, want 403", webErr.Status)
	}
}

func TestCSRFToken_BackwardCompat(t *testing.T) {
	cfg := testCSRFConfig()
	mw := CSRF(cfg)

	var capturedToken string
	handler := mw(func(c *web.Context) (web.Response, error) {
		capturedToken = CSRFToken(c)
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/test"})
	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if capturedToken == "" {
		t.Fatal("CSRFToken() returned empty string after context data struct change")
	}
}

func TestCSRFFieldName_ReturnsConfiguredName(t *testing.T) {
	cfg := testCSRFConfig()
	cfg.FormField = "custom_field"
	mw := CSRF(cfg)

	var capturedName string
	handler := mw(func(c *web.Context) (web.Response, error) {
		capturedName = CSRFFieldName(c)
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/test"})
	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if capturedName != "custom_field" {
		t.Errorf("CSRFFieldName() = %q, want %q", capturedName, "custom_field")
	}
}

func TestCSRFHeaderName_ReturnsConfiguredName(t *testing.T) {
	cfg := testCSRFConfig()
	cfg.HeaderName = "X-Custom"
	mw := CSRF(cfg)

	var capturedName string
	handler := mw(func(c *web.Context) (web.Response, error) {
		capturedName = CSRFHeaderName(c)
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/test"})
	resp, _ := handler(web.NewContext(context.Background(), req))

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if capturedName != "X-Custom" {
		t.Errorf("CSRFHeaderName() = %q, want %q", capturedName, "X-Custom")
	}
}

func TestCSRFToken_NoTokenInContext_ReturnsEmptyViaNewAccessors(t *testing.T) {
	// nil context: all accessors return zero values
	if got := CSRFFieldName(nil); got != "" {
		t.Errorf("CSRFFieldName(nil) = %q, want empty", got)
	}
	if got := CSRFHeaderName(nil); got != "" {
		t.Errorf("CSRFHeaderName(nil) = %q, want empty", got)
	}
}

func TestCSRF_UsesSessionBackedTokensWhenSessionMiddlewarePresent(t *testing.T) {
	codec, err := cookiecodec.New(cookiecodec.Config{
		Name:    "session",
		Secrets: [][]byte{[]byte("session-secret-012345678901234567890123")},
		Mode:    cookiecodec.Signed,
	})
	if err != nil {
		t.Fatalf("cookiecodec.New() error = %v", err)
	}

	_ = codec // codec configured above; MemoryStore uses random tokens, no codec required
	sessMW := websession.Middleware(websession.Config{
		Store: websession.NewMemoryStore(),
		CookieTemplate: web.Cookie{
			Name:     "session",
			Path:     "/",
			HTTPOnly: true,
			SameSite: "Lax",
		},
	})
	csrfMW := CSRF(testCSRFConfig())

	handler := sessMW(csrfMW(func(c *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, CSRFToken(c)), nil
	}))

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/test"})
	resp, _ := handler(web.NewContext(context.Background(), req))
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}

	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	token := string(tokenBytes)
	if len(token) != 64 {
		t.Fatalf("token length = %d, want 64", len(token))
	}
	if strings.Count(resp.Headers.Get("Set-Cookie"), "csrf=") != 0 {
		t.Fatalf("session-backed CSRF should not emit standalone csrf cookie, got %q", resp.Headers.Get("Set-Cookie"))
	}
}

func TestCSRF_AllowMissingOrigin_False_BlocksUnsafeRequest(t *testing.T) {
	cfg := testCSRFConfig()
	cfg.AllowMissingOrigin = false
	mw := CSRF(cfg)

	tok, _ := IssueToken(cfg.Key, "csrf", cfg.TokenTTL)
	handler := mw(func(c *web.Context) (web.Response, error) { return web.Respond(http.StatusOK), nil })

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/test"})
	req.Headers.Set("X-CSRF-Token", tok)
	req.Headers.Set("Cookie", "csrf="+tok)
	// No Origin or Referer — must be rejected when AllowMissingOrigin is false.

	_, err := handler(web.NewContext(context.Background(), req))
	assertForbidden(t, err, "CSRF origin invalid")
}

func TestCSRF_AllowMissingOrigin_True_AllowsUnsafeRequestWithoutOrigin(t *testing.T) {
	cfg := testCSRFConfig()
	cfg.AllowMissingOrigin = true
	mw := CSRF(cfg)

	tok, _ := IssueToken(cfg.Key, "csrf", cfg.TokenTTL)
	handler := mw(func(c *web.Context) (web.Response, error) { return web.Respond(http.StatusOK), nil })

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/test"})
	req.Headers.Set("X-CSRF-Token", tok)
	req.Headers.Set("Cookie", "csrf="+tok)
	// No Origin or Referer — must pass when AllowMissingOrigin is true.

	resp, _ := handler(web.NewContext(context.Background(), req))
	if resp.Status != http.StatusOK {
		t.Fatalf("AllowMissingOrigin=true: status = %d, want 200", resp.Status)
	}
}

func TestCSRF_CustomHeaderName_IsValidatedOnSubmission(t *testing.T) {
	cfg := testCSRFConfig()
	cfg.HeaderName = "X-Custom-Token"
	mw := CSRF(cfg)

	tok, err := IssueToken(cfg.Key, "csrf", cfg.TokenTTL)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/test"})
	req.Headers.Set("X-Custom-Token", tok)
	req.Headers.Set("Cookie", "csrf="+tok)

	resp, _ := handler(web.NewContext(context.Background(), req))
	if resp.Status != http.StatusOK {
		t.Fatalf("custom HeaderName: status = %d, want 200; token submitted via advertised header must be accepted", resp.Status)
	}
	if !called {
		t.Error("next handler was not called")
	}
}

func TestValidOrigin_UsesServerOriginOverHostHeader(t *testing.T) {
	cfg := testCSRFConfig()
	cfg.ServerOrigin = "https://legit.example.com"

	req := web.NewRequest(http.MethodPost, &url.URL{Scheme: "https", Host: "attacker.example.com", Path: "/submit"})
	req.Headers.Set("Origin", "https://legit.example.com")

	c := web.NewContext(context.Background(), req)
	got := validOrigin(c, cfg)
	if !got {
		t.Fatal("validOrigin() = false, want true: ServerOrigin should match Origin even with spoofed Host")
	}
}

func TestValidOrigin_ServerOriginRejectsMismatch(t *testing.T) {
	cfg := testCSRFConfig()
	cfg.ServerOrigin = "https://legit.example.com"

	req := web.NewRequest(http.MethodPost, &url.URL{Scheme: "https", Host: "attacker.example.com", Path: "/submit"})
	req.Headers.Set("Origin", "https://attacker.example.com")

	c := web.NewContext(context.Background(), req)
	got := validOrigin(c, cfg)
	if got {
		t.Fatal("validOrigin() = true, want false: Origin matching spoofed Host should be rejected when ServerOrigin differs")
	}
}

// TestSameOrigin_DefaultPortNotCanonicalized documents that sameOrigin performs
// strict string comparison on the Host field (including port). "https://example.com"
// and "https://example.com:443" are intentionally treated as distinct — port
// canonicalization would require tracking default ports per scheme. Change this
// test if that behavior is intentionally updated in the future.
func TestSameOrigin_DefaultPortNotCanonicalized(t *testing.T) {
	if sameOrigin("https://example.com", "https://example.com:443") {
		t.Error("sameOrigin matched default port — strict host comparison is expected; update this test if port canonicalization is added")
	}
}

func TestCSRF_AllowedOriginFunc_ReturnsTrue_Passes(t *testing.T) {
	cfg := testCSRFConfig()
	cfg.AllowMissingOrigin = false
	cfg.AllowedOriginFunc = func(origin string, _ *web.Context) bool {
		return origin == "http://dynamic.example.com"
	}
	mw := CSRF(cfg)

	tok, err := IssueToken(cfg.Key, "csrf", cfg.TokenTTL)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/submit"})
	req.Headers.Set("Origin", "http://dynamic.example.com")
	req.Headers.Set(cfg.HeaderName, tok)
	req.Headers.Set("Cookie", cfg.CookieName+"="+tok)

	resp, err := handler(web.NewContext(context.Background(), req))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d: AllowedOriginFunc returned true so request should pass", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler was not called when AllowedOriginFunc returned true")
	}
}

func TestCSRF_AllowedOriginFunc_ReturnsFalse_Returns403(t *testing.T) {
	cfg := testCSRFConfig()
	cfg.AllowMissingOrigin = false
	cfg.AllowedOriginFunc = func(_ string, _ *web.Context) bool {
		return false
	}
	mw := CSRF(cfg)

	tok, err := IssueToken(cfg.Key, "csrf", cfg.TokenTTL)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	called := false
	handler := mw(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodPost, &url.URL{Path: "/submit"})
	req.Headers.Set("Origin", "http://untrusted.example.com")
	req.Headers.Set(cfg.HeaderName, tok)
	req.Headers.Set("Cookie", cfg.CookieName+"="+tok)

	_, err = handler(web.NewContext(context.Background(), req))
	var webErr *web.Error
	if !errors.As(err, &webErr) {
		t.Fatalf("expected *web.Error when AllowedOriginFunc returns false, got %T: %v", err, err)
	}
	if webErr.Status != http.StatusForbidden {
		t.Errorf("status = %d, want %d", webErr.Status, http.StatusForbidden)
	}
	if called {
		t.Error("next handler was called when AllowedOriginFunc returned false")
	}
}
