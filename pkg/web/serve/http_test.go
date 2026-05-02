package serve

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/web"
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
		name       string
		setupReq   func(r *http.Request)
		wantScheme string
		wantHost   string
	}{
		{
			name:       "plain HTTP defaults to http scheme",
			setupReq:   func(r *http.Request) {},
			wantScheme: "http",
			wantHost:   "example.com",
		},
		{
			name: "X-Forwarded-Proto ignored without trusted proxies",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-Forwarded-Proto", "https")
			},
			wantScheme: "http",
			wantHost:   "example.com",
		},
		{
			name: "X-Forwarded-Proto HTTPS uppercase also ignored without trusted proxies",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-Forwarded-Proto", "HTTPS")
			},
			wantScheme: "http",
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

func TestParseTrustedProxies(t *testing.T) {
	cases := []struct {
		name    string
		input   []string
		wantLen int
		wantErr bool
	}{
		{name: "nil input returns empty slice", input: nil, wantLen: 0},
		{name: "valid IPv4 CIDR", input: []string{"10.0.0.0/8"}, wantLen: 1},
		{name: "multiple valid CIDRs", input: []string{"10.0.0.0/8", "192.168.0.0/16"}, wantLen: 2},
		{name: "valid IPv6 CIDR", input: []string{"::1/128"}, wantLen: 1},
		{name: "invalid CIDR returns error", input: []string{"not-a-cidr"}, wantErr: true},
		{name: "mixed valid then invalid returns error", input: []string{"10.0.0.0/8", "bad"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseTrustedProxies(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("ParseTrustedProxies() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseTrustedProxies() error = %v", err)
			}
			if len(got) != tc.wantLen {
				t.Errorf("len = %d, want %d", len(got), tc.wantLen)
			}
		})
	}
}

func TestNormalizeProxyIP(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		wantIP string
		wantOK bool
	}{
		{name: "bare IPv4", input: "192.168.1.1", wantIP: "192.168.1.1", wantOK: true},
		{name: "IPv4 with port", input: "192.168.1.1:8080", wantIP: "192.168.1.1", wantOK: true},
		{name: "bare IPv6 loopback", input: "::1", wantIP: "::1", wantOK: true},
		{name: "bracketed IPv6", input: "[::1]", wantIP: "::1", wantOK: true},
		{name: "bracketed IPv6 with port", input: "[::1]:8080", wantIP: "::1", wantOK: true},
		{name: "full IPv6 canonicalized", input: "2001:0db8:0000:0000:0000:0000:0000:0001", wantIP: "2001:db8::1", wantOK: true},
		{name: "IPv4-mapped IPv6 canonicalized to IPv4", input: "::ffff:192.0.2.1", wantIP: "192.0.2.1", wantOK: true},
		{name: "whitespace padded", input: "  10.0.0.1  ", wantIP: "10.0.0.1", wantOK: true},
		{name: "empty string", input: "", wantIP: "", wantOK: false},
		{name: "whitespace only", input: "   ", wantIP: "", wantOK: false},
		{name: "hostname not an IP", input: "proxy.example.com", wantIP: "", wantOK: false},
		{name: "garbage", input: "definitely-not-an-ip", wantIP: "", wantOK: false},
		{name: "hostname with port", input: "proxy.example.com:8080", wantIP: "", wantOK: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotIP, gotOK := NormalizeProxyIP(tc.input)
			if gotOK != tc.wantOK {
				t.Fatalf("NormalizeProxyIP(%q) ok = %v, want %v", tc.input, gotOK, tc.wantOK)
			}
			if gotIP != tc.wantIP {
				t.Errorf("NormalizeProxyIP(%q) = %q, want %q", tc.input, gotIP, tc.wantIP)
			}
		})
	}
}

func TestIsTrustedProxy(t *testing.T) {
	mustCIDR := func(s string) *net.IPNet {
		_, ipnet, err := net.ParseCIDR(s)
		if err != nil {
			panic(err)
		}
		return ipnet
	}
	cases := []struct {
		name       string
		remoteAddr string
		trusted    []*net.IPNet
		want       bool
	}{
		{name: "empty trusted list", remoteAddr: "10.0.0.1:1234", trusted: nil, want: false},
		{name: "IP in CIDR", remoteAddr: "10.0.0.1:1234", trusted: []*net.IPNet{mustCIDR("10.0.0.0/8")}, want: true},
		{name: "IP not in CIDR", remoteAddr: "172.16.0.1:1234", trusted: []*net.IPNet{mustCIDR("10.0.0.0/8")}, want: false},
		{name: "IPv6 in CIDR", remoteAddr: "[::1]:1234", trusted: []*net.IPNet{mustCIDR("::1/128")}, want: true},
		{name: "remoteAddr without port falls back to bare IP", remoteAddr: "10.0.0.1", trusted: []*net.IPNet{mustCIDR("10.0.0.0/8")}, want: true},
		{name: "unparseable remoteAddr", remoteAddr: "not-an-ip", trusted: []*net.IPNet{mustCIDR("10.0.0.0/8")}, want: false},
		{name: "IPv4-mapped IPv6 trusted via IPv4 CIDR", remoteAddr: "::ffff:192.0.2.1", trusted: []*net.IPNet{mustCIDR("192.0.2.0/24")}, want: true},
		{name: "bracketed IPv6 bare trusted", remoteAddr: "[::1]", trusted: []*net.IPNet{mustCIDR("::1/128")}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsTrustedProxy(tc.remoteAddr, tc.trusted); got != tc.want {
				t.Errorf("IsTrustedProxy(%q) = %v, want %v", tc.remoteAddr, got, tc.want)
			}
		})
	}
}

// TestFromHTTPRequest_TrustedProxyScheme verifies that X-Forwarded-Proto is
// accepted only when the remote peer is within a configured trusted CIDR.
// httptest.NewRequest sets RemoteAddr = "192.0.2.1:1234" by default.
func TestFromHTTPRequest_TrustedProxyScheme(t *testing.T) {
	trusted, err := ParseTrustedProxies([]string{"192.0.2.0/24"})
	if err != nil {
		t.Fatalf("ParseTrustedProxies: %v", err)
	}
	cfg := Config{TrustedProxies: trusted}

	cases := []struct {
		name       string
		proto      string
		wantScheme string
	}{
		{name: "trusted peer with X-Forwarded-Proto https", proto: "https", wantScheme: "https"},
		{name: "trusted peer with X-Forwarded-Proto HTTPS uppercase", proto: "HTTPS", wantScheme: "https"},
		{name: "trusted peer with X-Forwarded-Proto http stays http", proto: "http", wantScheme: "http"},
		{name: "trusted peer with no proto header defaults to http", proto: "", wantScheme: "http"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Host = "example.com"
			if tc.proto != "" {
				r.Header.Set("X-Forwarded-Proto", tc.proto)
			}
			req := fromHTTPRequestWithConfig(r, cfg)
			if req.URL.Scheme != tc.wantScheme {
				t.Errorf("scheme = %q, want %q", req.URL.Scheme, tc.wantScheme)
			}
		})
	}

	t.Run("untrusted peer X-Forwarded-Proto is rejected", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "203.0.113.9:4321" // outside 192.0.2.0/24
		r.Header.Set("X-Forwarded-Proto", "https")
		req := fromHTTPRequestWithConfig(r, cfg)
		if req.URL.Scheme != "http" {
			t.Errorf("scheme = %q, want %q (untrusted peer must be rejected)", req.URL.Scheme, "http")
		}
	})
}

func TestResolveScheme(t *testing.T) {
	trusted, err := ParseTrustedProxies([]string{"192.0.2.0/24"})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name       string
		tls        bool
		remoteAddr string
		xfpHeaders []string // each entry is a separate Add() call
		want       string
	}{
		{
			name: "TLS wins regardless of proxy",
			tls:  true,
			want: "https",
		},
		{
			name:       "trusted single https",
			remoteAddr: "192.0.2.1:1234",
			xfpHeaders: []string{"https"},
			want:       "https",
		},
		{
			name:       "trusted single http stays http",
			remoteAddr: "192.0.2.1:1234",
			xfpHeaders: []string{"http"},
			want:       "http",
		},
		{
			name:       "trusted padded uppercase accepted",
			remoteAddr: "192.0.2.1:1234",
			xfpHeaders: []string{"  HTTPS  "},
			want:       "https",
		},
		{
			name:       "trusted multi-header rejected",
			remoteAddr: "192.0.2.1:1234",
			xfpHeaders: []string{"https", "http"},
			want:       "http",
		},
		{
			name:       "trusted comma-separated rejected",
			remoteAddr: "192.0.2.1:1234",
			xfpHeaders: []string{"https, http"},
			want:       "http",
		},
		{
			name:       "trusted malformed proto rejected",
			remoteAddr: "192.0.2.1:1234",
			xfpHeaders: []string{"ftp"},
			want:       "http",
		},
		{
			name:       "trusted javascript proto rejected",
			remoteAddr: "192.0.2.1:1234",
			xfpHeaders: []string{"javascript"},
			want:       "http",
		},
		{
			name:       "trusted no header defaults to http",
			remoteAddr: "192.0.2.1:1234",
			xfpHeaders: nil,
			want:       "http",
		},
		{
			name:       "untrusted https ignored",
			remoteAddr: "203.0.113.9:1234",
			xfpHeaders: []string{"https"},
			want:       "http",
		},
		{
			name:       "no trusted proxies configured",
			remoteAddr: "192.0.2.1:1234",
			xfpHeaders: []string{"https"},
			want:       "http",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tc.remoteAddr
			if tc.tls {
				r.TLS = &tls.ConnectionState{}
			}
			// Clear any header set by httptest, then add the test values.
			r.Header.Del("X-Forwarded-Proto")
			for _, v := range tc.xfpHeaders {
				r.Header.Add("X-Forwarded-Proto", v)
			}
			var nets []*net.IPNet
			if tc.name != "no trusted proxies configured" {
				nets = trusted
			}
			got := resolveScheme(r, nets)
			if got != tc.want {
				t.Errorf("resolveScheme() = %q, want %q", got, tc.want)
			}
		})
	}
}
