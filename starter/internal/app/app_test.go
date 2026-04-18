package app_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-sum/foundry/internal/app"
	"github.com/go-sum/web/serve"
)

const testCSRFHexKey = "0000000000000000000000000000000000000000000000000000000000000001" // for-tests-only

func setupTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "testing")
	t.Setenv("SECURITY_CSRF_KEY", testCSRFHexKey)
	t.Setenv("SITE_BASE_URL", "http://test.local")

	dir, err := os.MkdirTemp("", "static-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	t.Setenv("TEST_STATIC_DIR", dir)
}

func mustNew(t *testing.T) *app.App {
	t.Helper()
	a, err := app.New(context.Background())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return a
}

func TestApp_Healthz_Returns200(t *testing.T) {
	setupTestEnv(t)
	h := serve.ToHTTPHandler(mustNew(t).Handler())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", string(body), "ok")
	}
}

func TestApp_UnknownPath_Returns404(t *testing.T) {
	setupTestEnv(t)
	h := serve.ToHTTPHandler(mustNew(t).Handler())
	req := httptest.NewRequest(http.MethodGet, "/definitely-does-not-exist", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestApp_GET_SetsRequestIDHeader(t *testing.T) {
	setupTestEnv(t)
	h := serve.ToHTTPHandler(mustNew(t).Handler())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header not set")
	}
}

// TestApp_GET_SetsSessionCookie verifies that the session middleware issues a
// session cookie on the first request. CSRF uses session-backed tokens when a
// session is present (no separate csrf double-submit cookie).
func TestApp_GET_SetsSessionCookie(t *testing.T) {
	setupTestEnv(t)
	h := serve.ToHTTPHandler(mustNew(t).Handler())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	resp := rec.Result()
	var found bool
	for _, c := range resp.Cookies() {
		if c.Name == "session" {
			found = true
			break
		}
	}
	if !found {
		t.Error("session cookie not set in response")
	}
}

func TestApp_GET_SetsSecurityHeaders(t *testing.T) {
	setupTestEnv(t)
	h := serve.ToHTTPHandler(mustNew(t).Handler())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
}
