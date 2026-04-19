package session

import (
	"testing"
	"time"
)

func TestDefaultSettings(t *testing.T) {
	got := DefaultSettings()
	if got.CookieName != "session" {
		t.Fatalf("CookieName = %q, want %q", got.CookieName, "session")
	}
	if got.IdleTTL != 30*time.Minute {
		t.Fatalf("IdleTTL = %v, want %v", got.IdleTTL, 30*time.Minute)
	}
	if got.AbsoluteTTL != 24*time.Hour {
		t.Fatalf("AbsoluteTTL = %v, want %v", got.AbsoluteTTL, 24*time.Hour)
	}
	if !got.CookieSecure {
		t.Fatal("CookieSecure = false, want true")
	}
}

func TestNewConfig(t *testing.T) {
	store := NewMemoryStore()
	t.Cleanup(store.Stop)

	cfg := NewConfig(Settings{
		CookieName:   "app-session",
		IdleTTL:      15 * time.Minute,
		AbsoluteTTL:  12 * time.Hour,
		CookieSecure: false,
	}, store)

	if cfg.Store != store {
		t.Fatal("Store mismatch")
	}
	if got, want := cfg.CookieTemplate.Name, "app-session"; got != want {
		t.Fatalf("CookieTemplate.Name = %q, want %q", got, want)
	}
	if got, want := cfg.CookieTemplate.Path, "/"; got != want {
		t.Fatalf("CookieTemplate.Path = %q, want %q", got, want)
	}
	if !cfg.CookieTemplate.HTTPOnly {
		t.Fatal("CookieTemplate.HTTPOnly = false, want true")
	}
	if got, want := cfg.CookieTemplate.SameSite, "Lax"; got != want {
		t.Fatalf("CookieTemplate.SameSite = %q, want %q", got, want)
	}
	if cfg.CookieTemplate.Secure {
		t.Fatal("CookieTemplate.Secure = true, want false")
	}
	if got, want := cfg.TTL, 12*time.Hour; got != want {
		t.Fatalf("TTL = %v, want %v", got, want)
	}
	if got, want := cfg.IdleTTL, 15*time.Minute; got != want {
		t.Fatalf("IdleTTL = %v, want %v", got, want)
	}
}
