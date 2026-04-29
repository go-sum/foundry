package proxy

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/web"
)

// newTestContext builds a *web.Context from method and path for use in handler tests.
func newTestContext(method, path string) *web.Context {
	u := &url.URL{Path: path}
	req := web.NewRequest(method, u)
	return web.NewContext(context.Background(), req)
}

// newTestContextWithHost builds a *web.Context and sets the Host header field.
func newTestContextWithHost(method, path, host string) *web.Context {
	u := &url.URL{Path: path}
	req := web.NewRequest(method, u)
	req.SetHost(host)
	return web.NewContext(context.Background(), req)
}

// newTestContextFull builds a *web.Context with host, remoteAddr, and optional
// incoming headers all set.
func newTestContextFull(method, path, host, remoteAddr string, incomingHeaders map[string]string) *web.Context {
	u := &url.URL{Path: path}
	req := web.NewRequest(method, u)
	req.SetHost(host)
	req.SetRemoteAddr(remoteAddr)
	for name, value := range incomingHeaders {
		req.Headers.Set(name, value)
	}
	return web.NewContext(context.Background(), req)
}

func TestReverse_ForwardsRequest(t *testing.T) {
	var gotMethod, gotPath string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	h := Reverse(target, Options{})
	c := newTestContext(http.MethodPost, "/api/test")
	resp, rerr := h(c)
	if rerr != nil {
		t.Fatalf("Reverse returned error: %v", rerr)
	}
	// Drain body to ensure upstream handler ran.
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if gotMethod != http.MethodPost {
		t.Errorf("upstream method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/api/test" {
		t.Errorf("upstream path = %q, want %q", gotPath, "/api/test")
	}
}

func TestReverse_ForwardsResponseStatus(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	h := Reverse(target, Options{})
	c := newTestContext(http.MethodGet, "/")
	resp, _ := h(c)
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if resp.Status != http.StatusCreated {
		t.Errorf("status = %d, want %d", resp.Status, http.StatusCreated)
	}
}

func TestReverse_ForwardsResponseBody(t *testing.T) {
	const wantBody = "hello from upstream"

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, wantBody)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	h := Reverse(target, Options{})
	c := newTestContext(http.MethodGet, "/data")
	resp, _ := h(c)
	if resp.Body == nil {
		t.Fatal("response body is nil")
	}
	defer resp.Body.Close() //nolint:errcheck

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(got) != wantBody {
		t.Errorf("body = %q, want %q", string(got), wantBody)
	}
}

func TestReverse_SetsXForwardedHost(t *testing.T) {
	var gotHeader string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Forwarded-Host")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	h := Reverse(target, Options{})
	c := newTestContextWithHost(http.MethodGet, "/", "example.com")
	resp, _ := h(c)
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if gotHeader != "example.com" {
		t.Errorf("X-Forwarded-Host = %q, want %q", gotHeader, "example.com")
	}
}

func TestReverse_SetsXForwardedFor(t *testing.T) {
	cases := []struct {
		name             string
		host             string
		remoteAddr       string
		incomingXFF      string // empty means no incoming X-Forwarded-For header
		wantXFF          string // empty means header must be absent
		wantHeaderAbsent bool
	}{
		{
			name:       "fresh XFF valid addr",
			host:       "ignored.example.com",
			remoteAddr: "192.168.1.1:12345",
			wantXFF:    "192.168.1.1",
		},
		{
			name:        "client-supplied XFF stripped, only remote IP forwarded",
			host:        "ignored.example.com",
			remoteAddr:  "10.0.0.2:9999",
			incomingXFF: "203.0.113.5",
			wantXFF:     "10.0.0.2",
		},
		{
			name:             "RemoteAddr empty",
			host:             "ignored.example.com",
			remoteAddr:       "",
			wantHeaderAbsent: true,
		},
		{
			name:       "RemoteAddr bare IP is accepted",
			host:       "ignored.example.com",
			remoteAddr: "192.168.1.1",
			wantXFF:    "192.168.1.1",
		},
		{
			name:       "Host header is not used for XFF",
			host:       "example.com",
			remoteAddr: "10.1.2.3:8080",
			wantXFF:    "10.1.2.3",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotXFF string
			var xffPresent bool

			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, xffPresent = r.Header["X-Forwarded-For"]
				gotXFF = r.Header.Get("X-Forwarded-For")
				w.WriteHeader(http.StatusOK)
			}))
			defer upstream.Close()

			target, err := url.Parse(upstream.URL)
			if err != nil {
				t.Fatalf("parse upstream URL: %v", err)
			}

			incomingHeaders := map[string]string{}
			if tc.incomingXFF != "" {
				incomingHeaders["X-Forwarded-For"] = tc.incomingXFF
			}

			h := Reverse(target, Options{})
			c := newTestContextFull(http.MethodGet, "/", tc.host, tc.remoteAddr, incomingHeaders)
			resp, _ := h(c)
			if resp.Body != nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}

			if tc.wantHeaderAbsent {
				if xffPresent {
					t.Errorf("X-Forwarded-For header present with value %q, want absent", gotXFF)
				}
				return
			}

			if !xffPresent {
				t.Errorf("X-Forwarded-For header absent, want %q", tc.wantXFF)
				return
			}
			if gotXFF != tc.wantXFF {
				t.Errorf("X-Forwarded-For = %q, want %q", gotXFF, tc.wantXFF)
			}
		})
	}
}

// recordingTransport wraps a delegate RoundTripper and records whether it was called.
type recordingTransport struct {
	called   bool
	delegate http.RoundTripper
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.called = true
	return rt.delegate.RoundTrip(req)
}

func TestReverse_CustomClient_IsUsed(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	transport := &recordingTransport{delegate: http.DefaultTransport}
	customClient := &http.Client{Transport: transport}

	h := Reverse(target, Options{Client: customClient})
	c := newTestContext(http.MethodGet, "/")
	resp, _ := h(c)
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if !transport.called {
		t.Error("custom client transport was not called; expected it to be used for the upstream request")
	}
}

func TestReverse_NilClient_UsesDefault(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	// Options{} has a nil Client field — the proxy must fall back to http.DefaultClient.
	h := Reverse(target, Options{})
	c := newTestContext(http.MethodGet, "/")
	resp, _ := h(c)
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if resp.Status != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.Status, http.StatusOK)
	}
}

func TestDynamicHopHeaders(t *testing.T) {
	cases := []struct {
		name       string
		connection string
		want       map[string]bool
	}{
		{
			name:       "empty header returns nil",
			connection: "",
			want:       nil,
		},
		{
			name:       "single token",
			connection: "X-Internal",
			want:       map[string]bool{"x-internal": true},
		},
		{
			name:       "multiple tokens",
			connection: "Keep-Alive, X-Custom, X-Backend",
			want:       map[string]bool{"keep-alive": true, "x-custom": true, "x-backend": true},
		},
		{
			name:       "lowercases names",
			connection: "UPGRADE",
			want:       map[string]bool{"upgrade": true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := dynamicHopHeaders(tc.connection)
			if tc.want == nil {
				if got != nil {
					t.Errorf("dynamicHopHeaders(%q) = %v, want nil", tc.connection, got)
				}
				return
			}
			for k := range tc.want {
				if !got[k] {
					t.Errorf("dynamicHopHeaders(%q) missing key %q", tc.connection, k)
				}
			}
			if len(got) != len(tc.want) {
				t.Errorf("dynamicHopHeaders(%q) len = %d, want %d", tc.connection, len(got), len(tc.want))
			}
		})
	}
}

func TestReverse_DynamicHopByHop_RequestHeaders(t *testing.T) {
	var gotHeaders http.Header

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, _ := url.Parse(upstream.URL)
	h := Reverse(target, Options{})

	c := newTestContext(http.MethodPost, "/")
	// Client sends Connection: X-Internal, naming X-Internal as hop-by-hop.
	c.Request.Headers.Set("Connection", "X-Internal")
	c.Request.Headers.Set("X-Internal", "secret")
	c.Request.Headers.Set("X-Normal", "visible")

	resp, _ := h(c)
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if gotHeaders.Get("X-Internal") != "" {
		t.Errorf("X-Internal forwarded to upstream, want stripped (dynamic hop-by-hop)")
	}
	if gotHeaders.Get("X-Normal") != "visible" {
		t.Errorf("X-Normal = %q, want %q", gotHeaders.Get("X-Normal"), "visible")
	}
}

func TestReverse_DynamicHopByHop_ResponseHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Upstream names X-Backend-Token as hop-by-hop via Connection.
		w.Header().Set("Connection", "X-Backend-Token")
		w.Header().Set("X-Backend-Token", "internal-value")
		w.Header().Set("X-Public", "visible")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, _ := url.Parse(upstream.URL)
	h := Reverse(target, Options{})

	c := newTestContext(http.MethodGet, "/")
	resp, _ := h(c)
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if resp.Headers.Get("X-Backend-Token") != "" {
		t.Errorf("X-Backend-Token present in response, want stripped (dynamic hop-by-hop)")
	}
	if resp.Headers.Get("X-Public") != "visible" {
		t.Errorf("X-Public = %q, want visible", resp.Headers.Get("X-Public"))
	}
}

func TestReverse_ForwardProto(t *testing.T) {
	var gotProto string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotProto = r.Header.Get("X-Forwarded-Proto")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, _ := url.Parse(upstream.URL)

	t.Run("sets header from URL scheme", func(t *testing.T) {
		h := Reverse(target, Options{})
		c := newTestContext(http.MethodGet, "/")
		// Simulate an HTTPS request by setting URL.Scheme on the context.
		c.Request.URL.Scheme = "https"

		resp, _ := h(c)
		if resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		if gotProto != "https" {
			t.Errorf("X-Forwarded-Proto = %q, want %q", gotProto, "https")
		}
	})

	t.Run("strips client header and sets from connection", func(t *testing.T) {
		gotProto = ""
		h := Reverse(target, Options{})
		c := newTestContext(http.MethodGet, "/")
		c.Request.Headers.Set("X-Forwarded-Proto", "https") // client injection attempt

		resp, _ := h(c)
		if resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		// Client-supplied value must be rejected; connection has no scheme so defaults to http.
		if gotProto != "http" {
			t.Errorf("X-Forwarded-Proto = %q, want %q (client injection rejected)", gotProto, "http")
		}
	})
}

func TestReverse_SetCookiePathRewrite(t *testing.T) {
	cases := []struct {
		name           string
		upstreamCookie string
		pathPrefix     string
		wantCookie     string
	}{
		{
			name:           "Path=/ rewritten to prefix",
			upstreamCookie: "id=1; Path=/; HttpOnly",
			pathPrefix:     "/app",
			wantCookie:     "id=1; Path=/app; HttpOnly",
		},
		{
			name:           "Path=/api rewritten with prefix prepended",
			upstreamCookie: "id=1; Path=/api; HttpOnly",
			pathPrefix:     "/app",
			wantCookie:     "id=1; Path=/app/api; HttpOnly",
		},
		{
			name:           "no PathPrefix leaves Path unchanged",
			upstreamCookie: "id=1; Path=/; HttpOnly",
			pathPrefix:     "",
			wantCookie:     "id=1; Path=/; HttpOnly",
		},
		{
			name:           "Domain and Path both rewritten",
			upstreamCookie: "id=1; Domain=upstream.internal; Path=/",
			pathPrefix:     "/app",
			wantCookie:     "id=1; Domain=public.example.com; Path=/app",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Set-Cookie", tc.upstreamCookie)
				w.WriteHeader(http.StatusOK)
			}))
			defer upstream.Close()

			target, _ := url.Parse(upstream.URL)
			h := Reverse(target, Options{PathPrefix: tc.pathPrefix})
			c := newTestContextFull(http.MethodGet, "/", "public.example.com", "10.0.0.1:1234", nil)
			resp, _ := h(c)
			if resp.Body != nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}

			cookies := resp.Headers.GetSetCookie()
			if len(cookies) == 0 {
				t.Fatal("Set-Cookie header absent")
			}
			if cookies[0] != tc.wantCookie {
				t.Errorf("Set-Cookie = %q, want %q", cookies[0], tc.wantCookie)
			}
		})
	}
}

func TestReverse_RewritesSetCookieDomain(t *testing.T) {
	cases := []struct {
		name           string
		upstreamCookie string
		incomingHost   string
		wantCookie     string
	}{
		{
			name:           "Domain present rewritten to public host",
			upstreamCookie: "session=abc; Domain=upstream.internal; Path=/",
			incomingHost:   "app.example.com",
			wantCookie:     "session=abc; Domain=app.example.com; Path=/",
		},
		{
			name:           "Public host has port port stripped",
			upstreamCookie: "session=abc; Domain=upstream.internal; Path=/",
			incomingHost:   "app.example.com:8443",
			wantCookie:     "session=abc; Domain=app.example.com; Path=/",
		},
		{
			name:           "No Domain attribute unchanged",
			upstreamCookie: "session=abc; Path=/",
			incomingHost:   "app.example.com",
			wantCookie:     "session=abc; Path=/",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Set-Cookie", tc.upstreamCookie)
				w.WriteHeader(http.StatusOK)
			}))
			defer upstream.Close()

			target, err := url.Parse(upstream.URL)
			if err != nil {
				t.Fatalf("parse upstream URL: %v", err)
			}

			h := Reverse(target, Options{})
			c := newTestContextFull(http.MethodGet, "/", tc.incomingHost, "10.0.0.1:1234", nil)
			resp, _ := h(c)
			if resp.Body != nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}

			cookies := resp.Headers.GetSetCookie()
			if len(cookies) == 0 {
				t.Fatalf("Set-Cookie header absent in response")
			}
			got := cookies[0]
			if got != tc.wantCookie {
				t.Errorf("Set-Cookie = %q, want %q", got, tc.wantCookie)
			}
		})
	}
}

func TestReverse_ClientTimeout_ReturnsBadGateway(t *testing.T) {
	// Upstream delays its response by 200ms; the client timeout is 50ms.
	// The proxy must return 502 Bad Gateway rather than hanging.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	shortClient := &http.Client{Timeout: 50 * time.Millisecond}
	h := Reverse(target, Options{Client: shortClient})
	c := newTestContext(http.MethodGet, "/")
	_, rerr := h(c)

	var webErr *web.Error
	if !errors.As(rerr, &webErr) {
		t.Fatalf("expected *web.Error on timeout, got %T: %v", rerr, rerr)
	}
	if webErr.Status != http.StatusBadGateway {
		t.Errorf("error status = %d, want %d (Bad Gateway on timeout)", webErr.Status, http.StatusBadGateway)
	}
}

func TestReverse_EmitsForwardedHeader(t *testing.T) {
	var gotForwarded string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotForwarded = r.Header.Get("Forwarded")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	h := Reverse(target, Options{})
	c := newTestContextFull(http.MethodGet, "/", "example.com", "203.0.113.5:4321", nil)
	resp, _ := h(c)
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if gotForwarded == "" {
		t.Fatal("Forwarded header absent, want non-empty")
	}
	if !strings.Contains(gotForwarded, "for=203.0.113.5") {
		t.Errorf("Forwarded = %q, want it to contain for=203.0.113.5", gotForwarded)
	}
	if !strings.Contains(gotForwarded, "host=example.com") {
		t.Errorf("Forwarded = %q, want it to contain host=example.com", gotForwarded)
	}
	if !strings.Contains(gotForwarded, "proto=http") {
		t.Errorf("Forwarded = %q, want it to contain proto=http", gotForwarded)
	}
}

func TestReverse_EmitsForwardedHeader_IPv6(t *testing.T) {
	var gotForwarded string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotForwarded = r.Header.Get("Forwarded")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	h := Reverse(target, Options{})
	// IPv6 RemoteAddr: brackets required by net package for host:port form.
	c := newTestContextFull(http.MethodGet, "/", "example.com", "[::1]:4321", nil)
	resp, _ := h(c)
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	// RFC 7239: IPv6 addresses must be quoted and bracketed: for="[::1]"
	if !strings.Contains(gotForwarded, `for="[::1]"`) {
		t.Errorf("Forwarded = %q, want it to contain for=\"[::1]\"", gotForwarded)
	}
}

// TestReverse_ClientForwardingHeaders_Stripped ensures that a client cannot inject
// fake forwarding metadata that would be trusted by upstream services.
func TestReverse_ClientForwardingHeaders_Stripped(t *testing.T) {
	cases := []struct {
		name   string
		header string
		value  string
	}{
		{"X-Forwarded-For injection", "X-Forwarded-For", "1.2.3.4"},
		{"Forwarded injection", "Forwarded", "for=1.2.3.4"},
		{"X-Forwarded-Host injection", "X-Forwarded-Host", "evil.example.com"},
		{"X-Forwarded-Proto injection", "X-Forwarded-Proto", "https"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotValue string
			var headerPresent bool

			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotValue = r.Header.Get(tc.header)
				_, headerPresent = r.Header[http.CanonicalHeaderKey(tc.header)]
				w.WriteHeader(http.StatusOK)
			}))
			defer upstream.Close()

			target, _ := url.Parse(upstream.URL)
			h := Reverse(target, Options{})
			// RemoteAddr is empty so X-Forwarded-For and Forwarded will be absent;
			// what matters is the client-injected value does not reach upstream.
			c := newTestContextFull(http.MethodGet, "/", "", "", map[string]string{
				tc.header: tc.value,
			})
			resp, _ := h(c)
			if resp.Body != nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}

			if headerPresent && gotValue == tc.value {
				t.Errorf("%s client-injected value %q reached upstream unchanged; want stripped", tc.header, tc.value)
			}
		})
	}
}

func TestReverse_HopByHopHeaders_Stripped(t *testing.T) {
	var gotHeaders http.Header

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, _ := url.Parse(upstream.URL)
	h := Reverse(target, Options{})

	c := newTestContext(http.MethodGet, "/")
	// Set static hop-by-hop headers that must be stripped.
	c.Request.Headers.Set("Connection", "keep-alive")
	c.Request.Headers.Set("Upgrade", "websocket")
	c.Request.Headers.Set("X-App-Header", "should-pass")

	resp, _ := h(c)
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if gotHeaders.Get("Upgrade") != "" {
		t.Errorf("Upgrade header forwarded to upstream, want stripped (hop-by-hop)")
	}
	if gotHeaders.Get("X-App-Header") != "should-pass" {
		t.Errorf("X-App-Header = %q, want %q", gotHeaders.Get("X-App-Header"), "should-pass")
	}
}

func TestReverse_TrustedProxy_ExtendsXFFChain(t *testing.T) {
	var gotXFF string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotXFF = r.Header.Get("X-Forwarded-For")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	_, trustedCIDR, _ := net.ParseCIDR("10.0.0.0/8")
	h := Reverse(target, Options{TrustedProxies: []*net.IPNet{trustedCIDR}})

	// RemoteAddr is in the trusted range; the LB already set X-Forwarded-For
	// for the end user. The proxy must extend the chain, not replace it.
	c := newTestContextFull(http.MethodGet, "/", "example.com", "10.0.0.1:1234", map[string]string{
		"X-Forwarded-For": "203.0.113.5",
	})
	resp, rerr := h(c)
	if rerr != nil {
		t.Fatalf("Reverse returned error: %v", rerr)
	}
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	want := "203.0.113.5, 10.0.0.1"
	if gotXFF != want {
		t.Errorf("X-Forwarded-For = %q, want %q", gotXFF, want)
	}
}

func TestReverse_UntrustedPeer_StartsXFFFresh(t *testing.T) {
	var gotXFF string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotXFF = r.Header.Get("X-Forwarded-For")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	_, trustedCIDR, _ := net.ParseCIDR("10.0.0.0/8")
	h := Reverse(target, Options{TrustedProxies: []*net.IPNet{trustedCIDR}})

	// RemoteAddr is 192.168.1.1 — NOT in 10.0.0.0/8. The inbound XFF must be
	// discarded and the chain must start fresh from RemoteAddr.
	c := newTestContextFull(http.MethodGet, "/", "example.com", "192.168.1.1:1234", map[string]string{
		"X-Forwarded-For": "203.0.113.5",
	})
	resp, rerr := h(c)
	if rerr != nil {
		t.Fatalf("Reverse returned error: %v", rerr)
	}
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	want := "192.168.1.1"
	if gotXFF != want {
		t.Errorf("X-Forwarded-For = %q, want %q (untrusted peer must start chain fresh)", gotXFF, want)
	}
}

func TestReverse_TrustedProxy_ExtendsForwardedChain(t *testing.T) {
	var gotForwarded string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotForwarded = r.Header.Get("Forwarded")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	_, trustedCIDR, _ := net.ParseCIDR("10.0.0.0/8")
	h := Reverse(target, Options{TrustedProxies: []*net.IPNet{trustedCIDR}})

	c := newTestContextFull(http.MethodGet, "/", "example.com", "10.0.0.1:1234", map[string]string{
		"Forwarded": "for=203.0.113.5;host=example.com;proto=https",
	})
	resp, rerr := h(c)
	if rerr != nil {
		t.Fatalf("Reverse returned error: %v", rerr)
	}
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if !strings.HasPrefix(gotForwarded, "for=203.0.113.5;host=example.com;proto=https, ") {
		t.Errorf("Forwarded = %q, want inbound chain preserved as prefix", gotForwarded)
	}
	if !strings.Contains(gotForwarded, "for=10.0.0.1") {
		t.Errorf("Forwarded = %q, want trusted peer hop appended", gotForwarded)
	}
}

func TestReverse_TrustedProxy_InvalidXFFStartsFresh(t *testing.T) {
	var gotXFF string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotXFF = r.Header.Get("X-Forwarded-For")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	_, trustedCIDR, _ := net.ParseCIDR("10.0.0.0/8")
	h := Reverse(target, Options{TrustedProxies: []*net.IPNet{trustedCIDR}})

	c := newTestContextFull(http.MethodGet, "/", "example.com", "10.0.0.1:1234", map[string]string{
		"X-Forwarded-For": "203.0.113.5, definitely-not-an-ip",
	})
	resp, rerr := h(c)
	if rerr != nil {
		t.Fatalf("Reverse returned error: %v", rerr)
	}
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if gotXFF != "10.0.0.1" {
		t.Errorf("X-Forwarded-For = %q, want %q when trusted peer supplies malformed chain", gotXFF, "10.0.0.1")
	}
}

func TestReverse_TrustedProxy_InvalidForwardedStartsFresh(t *testing.T) {
	var gotForwarded string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotForwarded = r.Header.Get("Forwarded")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	_, trustedCIDR, _ := net.ParseCIDR("10.0.0.0/8")
	h := Reverse(target, Options{TrustedProxies: []*net.IPNet{trustedCIDR}})

	c := newTestContextFull(http.MethodGet, "/", "example.com", "10.0.0.1:1234", map[string]string{
		"Forwarded": "for=203.0.113.5;host=example.com;proto=https, nope",
	})
	resp, rerr := h(c)
	if rerr != nil {
		t.Fatalf("Reverse returned error: %v", rerr)
	}
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if strings.Contains(gotForwarded, "203.0.113.5") {
		t.Errorf("Forwarded = %q, want malformed trusted chain dropped", gotForwarded)
	}
	if !strings.Contains(gotForwarded, "for=10.0.0.1") {
		t.Errorf("Forwarded = %q, want fresh hop emitted", gotForwarded)
	}
}


// --- X-Forwarded-Host trust-gated preservation ---

func TestReverse_XForwardedHost_AlwaysRebuiltFromHost(t *testing.T) {
	var gotXFH string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotXFH = r.Header.Get("X-Forwarded-Host")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, _ := url.Parse(upstream.URL)
	_, trustedCIDR, _ := net.ParseCIDR("10.0.0.0/8")
	h := Reverse(target, Options{TrustedProxies: []*net.IPNet{trustedCIDR}})

	// Even a trusted peer supplying X-Forwarded-Host must be ignored.
	// The outbound header must be built from c.Request.Host() (the actual Host header).
	c := newTestContextFull(http.MethodGet, "/", "internal.svc:8080", "10.0.0.1:1234", map[string]string{
		"X-Forwarded-Host": "public.example.com",
	})
	resp, err := h(c)
	if err != nil {
		t.Fatalf("Reverse error: %v", err)
	}
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if gotXFH != "internal.svc:8080" {
		t.Errorf("X-Forwarded-Host = %q, want %q (inbound XFH must never be preserved; rebuild from Host)", gotXFH, "internal.svc:8080")
	}
}

func TestReverse_UntrustedPeer_OverwritesXForwardedHost(t *testing.T) {
	var gotXFH string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotXFH = r.Header.Get("X-Forwarded-Host")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, _ := url.Parse(upstream.URL)
	_, trustedCIDR, _ := net.ParseCIDR("10.0.0.0/8")
	h := Reverse(target, Options{TrustedProxies: []*net.IPNet{trustedCIDR}})

	// RemoteAddr 192.168.1.1 is NOT in the trusted CIDR.
	c := newTestContextFull(http.MethodGet, "/", "example.com", "192.168.1.1:1234", map[string]string{
		"X-Forwarded-Host": "evil.example.com",
	})
	resp, err := h(c)
	if err != nil {
		t.Fatalf("Reverse error: %v", err)
	}
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if gotXFH != "example.com" {
		t.Errorf("X-Forwarded-Host = %q, want %q (untrusted peer injection must be replaced with Host)", gotXFH, "example.com")
	}
}

func TestReverse_Authority_OverridesRequestHost(t *testing.T) {
	var gotXFH, gotCookieDomain string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotXFH = r.Header.Get("X-Forwarded-Host")
		// Respond with a Set-Cookie that has a Domain attribute so we can
		// verify domain rewriting uses Authority, not the request Host.
		w.Header().Set("Set-Cookie", "sess=abc; Domain=internal.svc; Path=/")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, _ := url.Parse(upstream.URL)
	h := Reverse(target, Options{Authority: "public.example.com"})

	// Request Host is an internal service name that should NOT appear in outbound headers.
	c := newTestContextFull(http.MethodGet, "/", "internal.svc:8080", "10.0.0.1:1234", nil)
	resp, err := h(c)
	if err != nil {
		t.Fatalf("Reverse error: %v", err)
	}
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if gotXFH != "public.example.com" {
		t.Errorf("X-Forwarded-Host = %q, want %q", gotXFH, "public.example.com")
	}
	gotCookieDomain = resp.Headers.Get("Set-Cookie")
	if !strings.Contains(gotCookieDomain, "Domain=public.example.com") {
		t.Errorf("Set-Cookie = %q, want Domain=public.example.com", gotCookieDomain)
	}
}

func TestReverse_TrustedProxy_NoInboundXFH_FallsBackToHost(t *testing.T) {
	var gotXFH string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotXFH = r.Header.Get("X-Forwarded-Host")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, _ := url.Parse(upstream.URL)
	_, trustedCIDR, _ := net.ParseCIDR("10.0.0.0/8")
	h := Reverse(target, Options{TrustedProxies: []*net.IPNet{trustedCIDR}})

	c := newTestContextFull(http.MethodGet, "/", "example.com", "10.0.0.1:1234", nil)
	resp, err := h(c)
	if err != nil {
		t.Fatalf("Reverse error: %v", err)
	}
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if gotXFH != "example.com" {
		t.Errorf("X-Forwarded-Host = %q, want %q (trusted peer with no inbound XFH must fall back to Host)", gotXFH, "example.com")
	}
}

func TestReverse_NoTrustedProxies_XForwardedHostFromHost(t *testing.T) {
	var gotXFH string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotXFH = r.Header.Get("X-Forwarded-Host")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, _ := url.Parse(upstream.URL)
	h := Reverse(target, Options{}) // no TrustedProxies

	c := newTestContextFull(http.MethodGet, "/", "example.com", "10.0.0.1:1234", map[string]string{
		"X-Forwarded-Host": "injected.example.com",
	})
	resp, err := h(c)
	if err != nil {
		t.Fatalf("Reverse error: %v", err)
	}
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if gotXFH != "example.com" {
		t.Errorf("X-Forwarded-Host = %q, want %q (no trusted proxies configured, must use Host)", gotXFH, "example.com")
	}
}

// --- X-Forwarded-Proto trust-gated passthrough ---

func TestReverse_XForwardedProto_AlwaysRebuiltFromScheme(t *testing.T) {
	var gotProto string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotProto = r.Header.Get("X-Forwarded-Proto")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, _ := url.Parse(upstream.URL)
	_, trustedCIDR, _ := net.ParseCIDR("10.0.0.0/8")
	h := Reverse(target, Options{
		TrustedProxies: []*net.IPNet{trustedCIDR},
	})

	// Must use c.URL().Scheme, not inbound value, even from trusted peers.
	// Context URL has no scheme set so it defaults to "http".
	c := newTestContextFull(http.MethodGet, "/", "example.com", "10.0.0.1:1234", map[string]string{
		"X-Forwarded-Proto": "https", // inbound from trusted peer — must be ignored
	})
	resp, err := h(c)
	if err != nil {
		t.Fatalf("Reverse error: %v", err)
	}
	if resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}

	if gotProto != "http" {
		t.Errorf("X-Forwarded-Proto = %q, want %q (must use connection scheme, not inbound)", gotProto, "http")
	}
}

func TestReverse_XForwardedProto_AlwaysSet(t *testing.T) {
	t.Run("trusted peer", func(t *testing.T) {
		var gotProto string
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotProto = r.Header.Get("X-Forwarded-Proto")
			w.WriteHeader(http.StatusOK)
		}))
		defer upstream.Close()

		target, _ := url.Parse(upstream.URL)
		_, trustedCIDR, _ := net.ParseCIDR("10.0.0.0/8")
		h := Reverse(target, Options{TrustedProxies: []*net.IPNet{trustedCIDR}})

		c := newTestContextFull(http.MethodGet, "/", "example.com", "10.0.0.1:1234", nil)
		c.Request.URL.Scheme = "https"
		resp, err := h(c)
		if err != nil {
			t.Fatalf("Reverse error: %v", err)
		}
		if resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		if gotProto != "https" {
			t.Errorf("trusted peer: X-Forwarded-Proto = %q, want %q", gotProto, "https")
		}
	})

	t.Run("untrusted peer", func(t *testing.T) {
		var gotProto string
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotProto = r.Header.Get("X-Forwarded-Proto")
			w.WriteHeader(http.StatusOK)
		}))
		defer upstream.Close()

		target, _ := url.Parse(upstream.URL)
		_, trustedCIDR, _ := net.ParseCIDR("10.0.0.0/8")
		h := Reverse(target, Options{TrustedProxies: []*net.IPNet{trustedCIDR}})

		// RemoteAddr 192.168.1.1 is NOT in the trusted CIDR.
		c := newTestContextFull(http.MethodGet, "/", "example.com", "192.168.1.1:1234", nil)
		c.Request.URL.Scheme = "https"
		resp, err := h(c)
		if err != nil {
			t.Fatalf("Reverse error: %v", err)
		}
		if resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		if gotProto != "https" {
			t.Errorf("untrusted peer: X-Forwarded-Proto = %q, want %q", gotProto, "https")
		}
	})
}
