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

func TestBuildAllowedHosts(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		extra   string
		want    []string
	}{
		{"base URL only", "https://example.com", "", []string{"example.com"}},
		{"base URL with port", "https://example.com:8443/path", "", []string{"example.com"}},
		{"extra hosts only", "", "www.example.com, cdn.example.com", []string{"www.example.com", "cdn.example.com"}},
		{"base URL and extra", "https://example.com", "www.example.com", []string{"example.com", "www.example.com"}},
		{"empty inputs", "", "", nil},
		{"unparseable base URL", "://bad", "", nil},
		{"extra with empty entries", "https://example.com", " , ,alias.example.com", []string{"example.com", "alias.example.com"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAllowedHosts(tt.baseURL, tt.extra)
			if len(got) != len(tt.want) {
				t.Fatalf("BuildAllowedHosts() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("BuildAllowedHosts()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
