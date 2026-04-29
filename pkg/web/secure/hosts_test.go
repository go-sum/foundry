package secure

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-sum/foundry/pkg/web"
)

func TestAllowedHosts(t *testing.T) {
	cfg := AllowedHostsConfig{
		Hosts: []string{"example.com", "www.example.com", "::1"},
	}
	tests := []struct {
		name       string
		method     string
		host       string
		wantStatus int
	}{
		{"exact match passes", http.MethodGet, "example.com", http.StatusOK},
		{"with port passes", http.MethodGet, "example.com:8080", http.StatusOK},
		{"www subdomain passes", http.MethodGet, "www.example.com", http.StatusOK},
		{"case insensitive passes", http.MethodGet, "Example.COM", http.StatusOK},
		{"unknown host blocked", http.MethodGet, "evil.com", http.StatusMisdirectedRequest},
		{"empty host blocked", http.MethodGet, "", http.StatusMisdirectedRequest},
		{"subdomain not in list blocked", http.MethodGet, "sub.example.com", http.StatusMisdirectedRequest},
		{"ipv6 bare brackets passes", http.MethodGet, "[::1]", http.StatusOK},
		{"ipv6 with port passes", http.MethodGet, "[::1]:8080", http.StatusOK},
		{"ipv6 not in list blocked", http.MethodGet, "[2001:db8::1]", http.StatusMisdirectedRequest},
		// POST — middleware must apply to all methods
		{"POST exact match passes", http.MethodPost, "example.com", http.StatusOK},
		{"POST unknown host blocked", http.MethodPost, "evil.com", http.StatusMisdirectedRequest},
	}

	mw := AllowedHosts(cfg)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			handler := mw(func(c *web.Context) (web.Response, error) {
				called = true
				return web.Respond(http.StatusOK), nil
			})

			req := web.NewRequest(tc.method, &url.URL{Path: "/"})
			req.SetHost(tc.host)

			resp, err := handler(web.NewContext(context.Background(), req))

			if tc.wantStatus == http.StatusOK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if resp.Status != http.StatusOK {
					t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
				}
				if !called {
					t.Error("next handler was not called")
				}
			} else {
				var webErr *web.Error
				if !errors.As(err, &webErr) {
					t.Fatalf("expected *web.Error, got %T: %v", err, err)
				}
				if webErr.Status != tc.wantStatus {
					t.Fatalf("error status = %d, want %d", webErr.Status, tc.wantStatus)
				}
				if called {
					t.Error("next handler was called but should have been blocked")
				}
			}
		})
	}
}

func TestAllowedHosts_Skipper(t *testing.T) {
	cfg := AllowedHostsConfig{
		Hosts:   []string{"example.com"},
		Skipper: func(_ *web.Context) bool { return true },
	}

	called := false
	handler := AllowedHosts(cfg)(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.SetHost("evil.com")

	resp, err := handler(web.NewContext(context.Background(), req))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler was not called when skipper returned true")
	}
}

func TestAllowedHosts_EmptyList_PassesAll(t *testing.T) {
	cfg := AllowedHostsConfig{Hosts: []string{}}

	called := false
	handler := AllowedHosts(cfg)(func(c *web.Context) (web.Response, error) {
		called = true
		return web.Respond(http.StatusOK), nil
	})

	req := web.NewRequest(http.MethodGet, &url.URL{Path: "/"})
	req.SetHost("example.com")

	resp, err := handler(web.NewContext(context.Background(), req))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}
	if !called {
		t.Error("next handler was not called with empty allow list (should be no-op)")
	}
}
