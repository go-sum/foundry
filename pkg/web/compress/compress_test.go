package compress

import (
	"compress/flate"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/go-sum/web"
)

// makeContext builds a web.Context with the given Accept-Encoding header.
func makeContext(acceptEncoding string) *web.Context {
	h := web.NewHeaders()
	if acceptEncoding != "" {
		h.Set("Accept-Encoding", acceptEncoding)
	}
	req := web.Request{
		Method:  http.MethodGet,
		URL:     &url.URL{Path: "/"},
		Headers: h,
	}
	return web.NewContext(context.Background(), req)
}

// bodyString returns the body content of a response as a string, closing Body.
func bodyString(t *testing.T, resp web.Response) string {
	t.Helper()
	if resp.Body == nil {
		return ""
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	return string(data)
}

// decompressGzip decompresses gzip-encoded data.
func decompressGzip(t *testing.T, r io.Reader) string {
	t.Helper()
	gr, err := gzip.NewReader(r)
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	defer gr.Close()
	data, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("reading gzip: %v", err)
	}
	return string(data)
}

// decompressDeflate decompresses deflate-encoded data.
func decompressDeflate(t *testing.T, r io.Reader) string {
	t.Helper()
	fr := flate.NewReader(r)
	defer fr.Close()
	data, err := io.ReadAll(fr)
	if err != nil {
		t.Fatalf("reading deflate: %v", err)
	}
	return string(data)
}

// largeBody produces a string of length > 1024 bytes.
func largeBody() string {
	return strings.Repeat("Hello, World! This is compressible text content. ", 30)
}

// smallBody produces a string of length < 1024 bytes.
func smallBody() string {
	return "short"
}

// htmlHandler returns a handler that emits HTML with the given body text.
func htmlHandler(body string) web.Handler {
	return func(c *web.Context) (web.Response, error) {
		h := web.NewHeaders()
		h.Set("Content-Type", "text/html; charset=UTF-8")
		return web.Response{
			Status:  http.StatusOK,
			Headers: h,
			Body:    io.NopCloser(strings.NewReader(body)),
		}, nil
	}
}

// jsonHandler returns a handler that emits application/json.
func jsonHandler(body string) web.Handler {
	return func(c *web.Context) (web.Response, error) {
		h := web.NewHeaders()
		h.Set("Content-Type", "application/json")
		return web.Response{
			Status:  http.StatusOK,
			Headers: h,
			Body:    io.NopCloser(strings.NewReader(body)),
		}, nil
	}
}

func TestMiddleware(t *testing.T) {
	body := largeBody()

	tests := []struct {
		name           string
		acceptEncoding string
		handler        web.Handler
		wantEncoding   string // expected Content-Encoding, "" means absent
		wantVary       bool
		wantCompressed bool
		wantBody       string
	}{
		{
			name:           "gzip accepted compresses html",
			acceptEncoding: "gzip",
			handler:        htmlHandler(body),
			wantEncoding:   "gzip",
			wantVary:       true,
			wantCompressed: true,
			wantBody:       body,
		},
		{
			name:           "deflate accepted compresses html",
			acceptEncoding: "deflate",
			handler:        htmlHandler(body),
			wantEncoding:   "deflate",
			wantVary:       true,
			wantCompressed: true,
			wantBody:       body,
		},
		{
			name:           "identity accepted no compression",
			acceptEncoding: "identity",
			handler:        htmlHandler(body),
			wantEncoding:   "",
			wantVary:       false,
			wantCompressed: false,
			wantBody:       body,
		},
		{
			name:           "no accept-encoding header no compression",
			acceptEncoding: "",
			handler:        htmlHandler(body),
			wantEncoding:   "",
			wantVary:       false,
			wantCompressed: false,
			wantBody:       body,
		},
		{
			name:           "content-encoding already set skips compression",
			acceptEncoding: "gzip",
			handler: func(c *web.Context) (web.Response, error) {
				h := web.NewHeaders()
				h.Set("Content-Type", "text/html; charset=UTF-8")
				h.Set("Content-Encoding", "br")
				return web.Response{
					Status:  http.StatusOK,
					Headers: h,
					Body:    io.NopCloser(strings.NewReader(body)),
				}, nil
			},
			wantEncoding:   "br",
			wantVary:       false,
			wantCompressed: false,
			wantBody:       body,
		},
		{
			name:           "status 206 skips compression",
			acceptEncoding: "gzip",
			handler: func(c *web.Context) (web.Response, error) {
				h := web.NewHeaders()
				h.Set("Content-Type", "text/html; charset=UTF-8")
				return web.Response{
					Status:  http.StatusPartialContent,
					Headers: h,
					Body:    io.NopCloser(strings.NewReader(body)),
				}, nil
			},
			wantEncoding:   "",
			wantVary:       false,
			wantCompressed: false,
			wantBody:       body,
		},
		{
			name:           "image/png not compressible",
			acceptEncoding: "gzip",
			handler: func(c *web.Context) (web.Response, error) {
				h := web.NewHeaders()
				h.Set("Content-Type", "image/png")
				return web.Response{
					Status:  http.StatusOK,
					Headers: h,
					Body:    io.NopCloser(strings.NewReader(body)),
				}, nil
			},
			wantEncoding:   "",
			wantVary:       false,
			wantCompressed: false,
			wantBody:       body,
		},
		{
			name:           "body smaller than minsize no compression",
			acceptEncoding: "gzip",
			handler:        htmlHandler(smallBody()),
			wantEncoding:   "",
			wantVary:       false,
			wantCompressed: false,
			wantBody:       smallBody(),
		},
		{
			name:           "application/json compressed",
			acceptEncoding: "gzip",
			handler:        jsonHandler(body),
			wantEncoding:   "gzip",
			wantVary:       true,
			wantCompressed: true,
			wantBody:       body,
		},
		{
			name:           "text/html with charset prefix match",
			acceptEncoding: "gzip",
			handler:        htmlHandler(body),
			wantEncoding:   "gzip",
			wantVary:       true,
			wantCompressed: true,
			wantBody:       body,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := Middleware(Config{})
			c := makeContext(tt.acceptEncoding)
			resp, _ := mw(tt.handler)(c)

			gotEncoding := resp.Headers.Get("Content-Encoding")
			if gotEncoding != tt.wantEncoding {
				t.Errorf("Content-Encoding = %q, want %q", gotEncoding, tt.wantEncoding)
			}

			if tt.wantVary {
				vary := resp.Headers.Get("Vary")
				if !strings.Contains(strings.ToLower(vary), "accept-encoding") {
					t.Errorf("Vary = %q, want it to contain Accept-Encoding", vary)
				}
			} else {
				// Only check Vary absence when not expected — no-compression paths
				// don't set Vary.
				vary := resp.Headers.Get("Vary")
				if !tt.wantVary && strings.Contains(strings.ToLower(vary), "accept-encoding") && tt.wantEncoding == "" {
					t.Errorf("Vary = %q, expected no Accept-Encoding in Vary for non-compressed response", vary)
				}
			}

			if tt.wantCompressed {
				var got string
				switch tt.wantEncoding {
				case "gzip":
					got = decompressGzip(t, resp.Body)
					if resp.Body != nil {
						resp.Body.Close()
					}
				case "deflate":
					got = decompressDeflate(t, resp.Body)
					if resp.Body != nil {
						resp.Body.Close()
					}
				}
				if got != tt.wantBody {
					t.Errorf("decompressed body = %q, want %q", got, tt.wantBody)
				}
			} else {
				got := bodyString(t, resp)
				if got != tt.wantBody {
					t.Errorf("body = %q, want %q", got, tt.wantBody)
				}
			}
		})
	}
}

func TestMiddlewareVaryAppended(t *testing.T) {
	// Verify Vary: Accept-Encoding is appended when Vary already has a field.
	body := largeBody()
	handler := func(c *web.Context) (web.Response, error) {
		h := web.NewHeaders()
		h.Set("Content-Type", "text/html; charset=UTF-8")
		h.Set("Vary", "Cookie")
		return web.Response{
			Status:  http.StatusOK,
			Headers: h,
			Body:    io.NopCloser(strings.NewReader(body)),
		}, nil
	}

	mw := Middleware(Config{})
	c := makeContext("gzip")
	resp, _ := mw(handler)(c)
	defer resp.Body.Close()

	vary := resp.Headers.Get("Vary")
	if !strings.Contains(strings.ToLower(vary), "accept-encoding") {
		t.Errorf("Vary = %q, want Accept-Encoding to be present", vary)
	}
	if !strings.Contains(strings.ToLower(vary), "cookie") {
		t.Errorf("Vary = %q, want Cookie to still be present", vary)
	}
}

func TestIsCompressible(t *testing.T) {
	allowed := defaultAllowedTypes

	tests := []struct {
		ct   string
		want bool
	}{
		{"text/html; charset=UTF-8", true},
		{"text/plain", true},
		{"text/css", true},
		{"application/json", true},
		{"application/javascript", true},
		{"application/xml", true},
		{"application/xhtml+xml", true},
		{"image/svg+xml", true},
		{"image/png", false},
		{"image/jpeg", false},
		{"application/octet-stream", false},
		{"video/mp4", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ct, func(t *testing.T) {
			got := isCompressible(tt.ct, allowed)
			if got != tt.want {
				t.Errorf("isCompressible(%q) = %v, want %v", tt.ct, got, tt.want)
			}
		})
	}
}

func TestMiddlewareNilBody(t *testing.T) {
	// Handler returns a nil body — middleware must not panic.
	handler := func(c *web.Context) (web.Response, error) {
		h := web.NewHeaders()
		h.Set("Content-Type", "text/html")
		return web.Response{
			Status:  http.StatusOK,
			Headers: h,
			Body:    nil,
		}, nil
	}

	mw := Middleware(Config{})
	c := makeContext("gzip")
	resp, _ := mw(handler)(c)

	if resp.Headers.Get("Content-Encoding") != "" {
		t.Errorf("Content-Encoding should not be set for nil body")
	}
}
