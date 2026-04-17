package adapt

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-sum/web"
)

func TestToHTTPHandler(t *testing.T) {
	h := func(_ *web.Context) (web.Response, error) {
		resp := web.Text(http.StatusOK, "hello")
		resp.Headers.Append("Set-Cookie", "a=1")
		resp.Headers.Append("Set-Cookie", "b=2")
		return resp, nil
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ToHTTPHandler(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "hello" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "hello")
	}
	cookies := rec.Result().Header.Values("Set-Cookie")
	if len(cookies) != 2 {
		t.Fatalf("Set-Cookie count = %d, want 2", len(cookies))
	}
}

func TestToHTTPHandlerWithConfigMaxBodyBytes(t *testing.T) {
	h := func(c *web.Context) (web.Response, error) {
		if _, err := c.Request.Bytes(); err != nil {
			return web.Text(http.StatusRequestEntityTooLarge, "too large"), nil
		}
		return web.Text(http.StatusOK, "ok"), nil
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("0123456789"))
	ToHTTPHandlerWithConfig(h, Config{MaxRequestBodyBytes: 5}).ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestWriteHTTPResponseHeadSkipsBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "/", nil)
	WriteHTTPResponse(rec, req, web.Text(http.StatusOK, "hello"), Config{})

	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rec.Body.String())
	}
}

func TestFromHTTPRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/users?page=1", strings.NewReader("hello"))
	r.Header.Set("X-Test", "value")
	r.RemoteAddr = "192.168.1.1:12345"

	input := FromHTTPRequest(r)
	if input.Method != http.MethodPost {
		t.Fatalf("method = %q, want %q", input.Method, http.MethodPost)
	}
	if input.URL.Path != "/api/users" {
		t.Fatalf("path = %q, want %q", input.URL.Path, "/api/users")
	}
	if input.Headers.Get("X-Test") != "value" {
		t.Fatalf("X-Test = %q", input.Headers.Get("X-Test"))
	}
	if input.RemoteAddr() != "192.168.1.1:12345" {
		t.Fatalf("remote addr = %q", input.RemoteAddr())
	}
}

func TestFromHTTPRequest_AbsoluteURL(t *testing.T) {
	cases := []struct {
		name        string
		setupReq    func(r *http.Request)
		wantScheme  string
		wantHost    string
	}{
		{
			name:       "plain HTTP defaults to http scheme",
			setupReq:   func(r *http.Request) {},
			wantScheme: "http",
			wantHost:   "example.com",
		},
		{
			name: "X-Forwarded-Proto https sets https scheme",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-Forwarded-Proto", "https")
			},
			wantScheme: "https",
			wantHost:   "example.com",
		},
		{
			name: "X-Forwarded-Proto case-insensitive",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-Forwarded-Proto", "HTTPS")
			},
			wantScheme: "https",
			wantHost:   "example.com",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/path", nil)
			r.Host = "example.com"
			tc.setupReq(r)

			input := FromHTTPRequest(r)

			if input.URL == nil {
				t.Fatal("URL is nil")
			}
			if input.URL.Scheme != tc.wantScheme {
				t.Errorf("URL.Scheme = %q, want %q", input.URL.Scheme, tc.wantScheme)
			}
			if input.URL.Host != tc.wantHost {
				t.Errorf("URL.Host = %q, want %q", input.URL.Host, tc.wantHost)
			}
			if input.URL.Path != "/path" {
				t.Errorf("URL.Path = %q, want %q", input.URL.Path, "/path")
			}
		})
	}
}

func TestWriteHTTPResponse_DropsTransportControlledHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	resp := web.Respond(http.StatusOK)
	resp.Headers.Set("Connection", "close")
	resp.Headers.Set("Transfer-Encoding", "chunked")
	resp.Headers.Set("Keep-Alive", "timeout=5")
	resp.Headers.Set("Upgrade", "websocket")
	resp.Headers.Set("Trailer", "Foo")
	resp.Headers.Set("X-Test", "ok")

	var warnings []error
	WriteHTTPResponse(rec, req, resp, Config{
		OnError: func(err error) {
			warnings = append(warnings, err)
		},
	})

	result := rec.Result()
	if got := result.Header.Get("Connection"); got != "" {
		t.Fatalf("Connection = %q, want empty", got)
	}
	if got := result.Header.Get("Transfer-Encoding"); got != "" {
		t.Fatalf("Transfer-Encoding = %q, want empty", got)
	}
	if got := result.Header.Get("Keep-Alive"); got != "" {
		t.Fatalf("Keep-Alive = %q, want empty", got)
	}
	if got := result.Header.Get("Upgrade"); got != "" {
		t.Fatalf("Upgrade = %q, want empty", got)
	}
	if got := result.Header.Get("Trailer"); got != "" {
		t.Fatalf("Trailer = %q, want empty", got)
	}
	if got := result.Header.Get("X-Test"); got != "ok" {
		t.Fatalf("X-Test = %q, want ok", got)
	}
	if len(warnings) != 5 {
		t.Fatalf("warning count = %d, want 5", len(warnings))
	}
	if got := warnings[0].Error(); !strings.Contains(got, "dropping transport-controlled response header") {
		t.Fatalf("warning = %q, want drop message", got)
	}
}
