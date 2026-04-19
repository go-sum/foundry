package site

import "testing"

func TestDefaultConfig(t *testing.T) {
	got := DefaultConfig()
	if got.BaseURL != "" {
		t.Fatalf("BaseURL = %q, want empty", got.BaseURL)
	}
	if got.OriginAllowlist != nil {
		t.Fatalf("OriginAllowlist = %#v, want nil", got.OriginAllowlist)
	}
}

func TestSiteHelpers(t *testing.T) {
	s := New(Config{
		BaseURL:         "https://example.com/app",
		OriginAllowlist: []string{"https://admin.example.com"},
	})

	if got, want := s.Origin(), "https://example.com"; got != want {
		t.Fatalf("Origin() = %q, want %q", got, want)
	}
	if got, want := s.AbsoluteURL("docs/getting-started"), "https://example.com/app/docs/getting-started"; got != want {
		t.Fatalf("AbsoluteURL() = %q, want %q", got, want)
	}
	if !s.IsAllowedOrigin("https://example.com") {
		t.Fatal("IsAllowedOrigin(self) = false, want true")
	}
	if !s.IsAllowedOrigin("https://admin.example.com") {
		t.Fatal("IsAllowedOrigin(allowlist) = false, want true")
	}
	if s.IsAllowedOrigin("https://evil.example.com") {
		t.Fatal("IsAllowedOrigin(untrusted) = true, want false")
	}
}

func TestNew_PanicsOnInvalidBaseURL(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for invalid BaseURL")
		}
	}()
	New(Config{BaseURL: "://bad-url"})
}
