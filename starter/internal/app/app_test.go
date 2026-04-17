package app_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-sum/foundry/internal/app"
	"github.com/go-sum/web/adapt"
)

const testCSRFHexKey = "0000000000000000000000000000000000000000000000000000000000000001" // for-tests-only

func setupTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "testing")
	t.Setenv("SECURITY_CSRF_KEY", testCSRFHexKey)
}

func TestApp_Healthz_Returns200(t *testing.T) {
	setupTestEnv(t)
	h := adapt.ToHTTPHandler(app.New().Handler())
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
	h := adapt.ToHTTPHandler(app.New().Handler())
	req := httptest.NewRequest(http.MethodGet, "/definitely-does-not-exist", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestApp_GET_SetsRequestIDHeader(t *testing.T) {
	setupTestEnv(t)
	h := adapt.ToHTTPHandler(app.New().Handler())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header not set")
	}
}

func TestApp_GET_SetsCookieCSRF(t *testing.T) {
	setupTestEnv(t)
	h := adapt.ToHTTPHandler(app.New().Handler())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	resp := rec.Result()
	var found bool
	for _, c := range resp.Cookies() {
		if c.Name == "csrf" {
			found = true
			break
		}
	}
	if !found {
		t.Error("csrf cookie not set in response")
	}
}

func TestApp_GET_SetsSecurityHeaders(t *testing.T) {
	setupTestEnv(t)
	h := adapt.ToHTTPHandler(app.New().Handler())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
}
