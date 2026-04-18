package site

import (
	"testing"
)

func TestSite_Origin(t *testing.T) {
	s := New(Config{BaseURL: "https://example.com/app"})
	if got := s.Origin(); got != "https://example.com" {
		t.Fatalf("Origin() = %q, want %q", got, "https://example.com")
	}
}

func TestSite_Origin_WithPort(t *testing.T) {
	s := New(Config{BaseURL: "http://localhost:8080"})
	if got := s.Origin(); got != "http://localhost:8080" {
		t.Fatalf("Origin() = %q, want %q", got, "http://localhost:8080")
	}
}

func TestSite_AbsoluteURL_NoLeadingSlash(t *testing.T) {
	s := New(Config{BaseURL: "https://example.com"})
	got := s.AbsoluteURL("about")
	if got != "https://example.com/about" {
		t.Fatalf("AbsoluteURL(%q) = %q, want %q", "about", got, "https://example.com/about")
	}
}

func TestSite_AbsoluteURL_WithLeadingSlash(t *testing.T) {
	s := New(Config{BaseURL: "https://example.com"})
	got := s.AbsoluteURL("/about")
	if got != "https://example.com/about" {
		t.Fatalf("AbsoluteURL(%q) = %q, want %q", "/about", got, "https://example.com/about")
	}
}

func TestSite_AbsoluteURL_BaseWithTrailingSlash(t *testing.T) {
	s := New(Config{BaseURL: "https://example.com/"})
	got := s.AbsoluteURL("/about")
	if got != "https://example.com/about" {
		t.Fatalf("AbsoluteURL(%q) = %q, want %q", "/about", got, "https://example.com/about")
	}
}

func TestSite_IsAllowedOrigin_OwnOrigin(t *testing.T) {
	s := New(Config{BaseURL: "https://example.com"})
	if !s.IsAllowedOrigin("https://example.com") {
		t.Fatal("own origin should be allowed")
	}
}

func TestSite_IsAllowedOrigin_AllowlistMatch(t *testing.T) {
	s := New(Config{
		BaseURL:         "https://example.com",
		OriginAllowlist: []string{"https://partner.com", "https://other.com"},
	})
	if !s.IsAllowedOrigin("https://partner.com") {
		t.Fatal("allowlisted origin should be allowed")
	}
	if !s.IsAllowedOrigin("https://other.com") {
		t.Fatal("allowlisted origin should be allowed")
	}
}

func TestSite_IsAllowedOrigin_Disallowed(t *testing.T) {
	s := New(Config{
		BaseURL:         "https://example.com",
		OriginAllowlist: []string{"https://partner.com"},
	})
	if s.IsAllowedOrigin("https://evil.com") {
		t.Fatal("non-allowlisted origin should not be allowed")
	}
}

func TestNew_InvalidBaseURL_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid BaseURL, got none")
		}
	}()
	New(Config{BaseURL: "not-a-url"})
}

func TestNew_EmptyBaseURL_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for empty BaseURL, got none")
		}
	}()
	New(Config{BaseURL: ""})
}
