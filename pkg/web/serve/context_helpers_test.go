package serve

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/web"
)

func TestToHTTPHandler_FallbackProblemResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/widgets", nil)

	ToHTTPHandler(func(_ *web.Context) (web.Response, error) {
		return web.Response{}, errors.New("boom")
	}).ServeHTTP(rec, req)

	if got, want := rec.Code, http.StatusInternalServerError; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if got, want := rec.Header().Get("Content-Type"), "application/problem+json"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got, want := body["title"], "Internal Server Error"; got != want {
		t.Fatalf("title = %#v, want %#v", got, want)
	}
	if got, want := body["instance"], "/widgets"; got != want {
		t.Fatalf("instance = %#v, want %#v", got, want)
	}
	if _, ok := body["detail"]; ok {
		t.Fatalf("detail present in 500 problem body: %#v", body["detail"])
	}
}

func TestNewContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "https://example.com/form", strings.NewReader("hello"))
	req.Header.Set("X-Test", "value")

	c := NewContext(req)
	if got, want := c.Method(), http.MethodPost; got != want {
		t.Fatalf("Method() = %q, want %q", got, want)
	}
	if got, want := c.URL().String(), "https://example.com/form"; got != want {
		t.Fatalf("URL() = %q, want %q", got, want)
	}
	if got, want := c.Headers().Get("X-Test"), "value"; got != want {
		t.Fatalf("X-Test = %q, want %q", got, want)
	}
}

func TestNewContextWithConfig_AppliesBodyLimit(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("abcdef"))

	c := NewContextWithConfig(rec, req, Config{MaxRequestBodyBytes: 3})
	_, err := c.Request.Bytes()
	if !errors.Is(err, web.ErrBodyTooLarge) {
		t.Fatalf("Bytes() error = %v, want ErrBodyTooLarge", err)
	}
}
