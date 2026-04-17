package etag

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/go-sum/web"
)

// hashOf returns the first 32 hex chars of the SHA-256 hash of body.
func hashOf(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])[:32]
}

func newContext(method string, headers map[string]string) *web.Context {
	req := web.NewRequest(method, &url.URL{Path: "/"})
	for k, v := range headers {
		req.Headers.Set(k, v)
	}
	return web.NewContext(context.Background(), req)
}

func bodyResponse(status int, body string) web.Response {
	resp := web.Respond(status)
	resp.Body = io.NopCloser(strings.NewReader(body))
	return resp
}

func TestMiddleware(t *testing.T) {
	const testBody = "Hello, ETag world!"
	strongTag := `"` + hashOf([]byte(testBody)) + `"`
	weakTag := `W/"` + hashOf([]byte(testBody)) + `"`

	tests := []struct {
		name           string
		cfg            Config
		reqHeaders     map[string]string
		handlerResp    func() web.Response
		wantStatus     int
		wantETag       string
		wantBodyEmpty  bool
		wantBodyEquals string
		wantNoETag     bool
	}{
		{
			name: "200 with small body sets strong ETag",
			cfg:  Config{},
			handlerResp: func() web.Response {
				return bodyResponse(http.StatusOK, testBody)
			},
			wantStatus:     http.StatusOK,
			wantETag:       strongTag,
			wantBodyEquals: testBody,
		},
		{
			name: "matching If-None-Match returns 304 empty body",
			cfg:  Config{},
			reqHeaders: map[string]string{
				"If-None-Match": strongTag,
			},
			handlerResp: func() web.Response {
				return bodyResponse(http.StatusOK, testBody)
			},
			wantStatus:    http.StatusNotModified,
			wantBodyEmpty: true,
			wantNoETag:    true,
		},
		{
			name: "If-None-Match wildcard returns 304",
			cfg:  Config{},
			reqHeaders: map[string]string{
				"If-None-Match": "*",
			},
			handlerResp: func() web.Response {
				return bodyResponse(http.StatusOK, testBody)
			},
			wantStatus:    http.StatusNotModified,
			wantBodyEmpty: true,
			wantNoETag:    true,
		},
		{
			name: "response already has ETag is untouched",
			cfg:  Config{},
			handlerResp: func() web.Response {
				resp := bodyResponse(http.StatusOK, testBody)
				resp.Headers.Set("ETag", `"existing"`)
				return resp
			},
			wantStatus: http.StatusOK,
			wantETag:   `"existing"`,
		},
		{
			name: "non-200 response 201 does not get ETag",
			cfg:  Config{},
			handlerResp: func() web.Response {
				return bodyResponse(http.StatusCreated, testBody)
			},
			wantStatus: http.StatusCreated,
			wantNoETag: true,
		},
		{
			name: "non-200 response 302 does not get ETag",
			cfg:  Config{},
			handlerResp: func() web.Response {
				return web.Redirect(http.StatusFound, "/other")
			},
			wantStatus: http.StatusFound,
			wantNoETag: true,
		},
		{
			name: "nil body does not get ETag",
			cfg:  Config{},
			handlerResp: func() web.Response {
				resp := web.Respond(http.StatusOK)
				// Body is nil by default from Respond.
				return resp
			},
			wantStatus: http.StatusOK,
			wantNoETag: true,
		},
		{
			name: "body larger than MaxBuffer streams through without ETag",
			cfg:  Config{MaxBuffer: 4},
			handlerResp: func() web.Response {
				return bodyResponse(http.StatusOK, "Hello, world!") // 13 bytes > 4
			},
			wantStatus:     http.StatusOK,
			wantNoETag:     true,
			wantBodyEquals: "Hello, world!",
		},
		{
			name: "weak ETag config emits W/ prefix",
			cfg:  Config{Weak: true},
			handlerResp: func() web.Response {
				return bodyResponse(http.StatusOK, testBody)
			},
			wantStatus:     http.StatusOK,
			wantETag:       weakTag,
			wantBodyEquals: testBody,
		},
		{
			name: "mismatched If-None-Match returns 200 with ETag and full body",
			cfg:  Config{},
			reqHeaders: map[string]string{
				"If-None-Match": `"different-tag-value-here-000000"`,
			},
			handlerResp: func() web.Response {
				return bodyResponse(http.StatusOK, testBody)
			},
			wantStatus:     http.StatusOK,
			wantETag:       strongTag,
			wantBodyEquals: testBody,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := Middleware(tt.cfg)

			c := newContext(http.MethodGet, tt.reqHeaders)

			resp, _ := mw(func(_ *web.Context) (web.Response, error) {
				return tt.handlerResp(), nil
			})(c)

			if resp.Status != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.Status, tt.wantStatus)
			}

			gotETag := resp.Headers.Get("ETag")

			if tt.wantNoETag && gotETag != "" {
				t.Errorf("ETag = %q, want none", gotETag)
			}

			if tt.wantETag != "" && gotETag != tt.wantETag {
				t.Errorf("ETag = %q, want %q", gotETag, tt.wantETag)
			}

			if tt.wantBodyEmpty {
				if resp.Body != nil {
					data, _ := io.ReadAll(resp.Body)
					if len(data) != 0 {
						t.Errorf("body = %q, want empty", data)
					}
				}
			}

			if tt.wantBodyEquals != "" {
				if resp.Body == nil {
					t.Fatal("body is nil, want non-nil")
				}
				data, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("reading body: %v", err)
				}
				if string(data) != tt.wantBodyEquals {
					t.Errorf("body = %q, want %q", data, tt.wantBodyEquals)
				}
			}
		})
	}
}

func TestMiddleware_DefaultMaxBuffer(t *testing.T) {
	// Verify that zero MaxBuffer defaults to 1 MiB and does not skip small bodies.
	mw := Middleware(Config{MaxBuffer: 0})

	body := "small body"
	c := newContext(http.MethodGet, nil)

	resp, _ := mw(func(_ *web.Context) (web.Response, error) {
		return bodyResponse(http.StatusOK, body), nil
	})(c)

	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if resp.Headers.Get("ETag") == "" {
		t.Error("ETag should be set for small body with default MaxBuffer")
	}
}

func TestMiddleware_LargeBodyFullyReadable(t *testing.T) {
	// Body larger than MaxBuffer must be fully readable from the reconstructed reader.
	const maxBuf = 8
	bigBody := bytes.Repeat([]byte("x"), 20)

	mw := Middleware(Config{MaxBuffer: maxBuf})
	c := newContext(http.MethodGet, nil)

	resp, _ := mw(func(_ *web.Context) (web.Response, error) {
		r := io.NopCloser(bytes.NewReader(bigBody))
		resp := web.Respond(http.StatusOK)
		resp.Body = r
		return resp, nil
	})(c)

	if resp.Headers.Get("ETag") != "" {
		t.Error("ETag should not be set when body exceeds MaxBuffer")
	}

	if resp.Body == nil {
		t.Fatal("body must not be nil after streaming through")
	}
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if !bytes.Equal(got, bigBody) {
		t.Errorf("body = %q (%d bytes), want %q (%d bytes)", got, len(got), bigBody, len(bigBody))
	}
}

func TestMiddleware_WeakETagMatchesOnIfNoneMatch(t *testing.T) {
	// A weak ETag in the response should match a weak ETag in If-None-Match.
	const body = "content"
	weakTag := `W/"` + hashOf([]byte(body)) + `"`

	mw := Middleware(Config{Weak: true})
	c := newContext(http.MethodGet, map[string]string{
		"If-None-Match": weakTag,
	})

	resp, _ := mw(func(_ *web.Context) (web.Response, error) {
		return bodyResponse(http.StatusOK, body), nil
	})(c)

	if resp.Status != http.StatusNotModified {
		t.Errorf("status = %d, want %d", resp.Status, http.StatusNotModified)
	}
}
