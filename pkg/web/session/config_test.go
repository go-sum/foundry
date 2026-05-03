package session

import (
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
)

func TestInitialSessionSettings(t *testing.T) {
	got := InitialSessionSettings("")
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
	if got.KVPrefix != "session:" {
		t.Fatalf("KVPrefix = %q, want %q", got.KVPrefix, "session:")
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

func TestNewCookieCodec(t *testing.T) {
	codec, err := NewCookieCodec(Settings{
		CookieName: "app-session",
		CookieKey:  []byte("01234567890123456789012345678901"),
	})
	if err != nil {
		t.Fatalf("CookieCodecFromSettings() error = %v", err)
	}
	if codec == nil {
		t.Fatal("CookieCodecFromSettings() = nil, want non-nil codec")
	}
}

func TestValidationRules_KV_RequiresPasswordOutsideTesting(t *testing.T) {
	for _, env := range []string{"production", "development"} {
		t.Run(env, func(t *testing.T) {
			v := validator.New()
			ValidationRules(StoreTypeKV, env, "", nil)(v)
			err := v.Struct(Settings{CookieName: "session"})
			if err == nil {
				t.Fatal("expected error for kv store without password, got nil")
			}
		})
	}
}

func TestValidationRules_Memory_OnlyAllowedInTesting(t *testing.T) {
	for _, env := range []string{"production", "development"} {
		t.Run(env, func(t *testing.T) {
			v := validator.New()
			ValidationRules(StoreTypeMemory, env, "", nil)(v)
			err := v.Struct(Settings{CookieName: "session"})
			if err == nil {
				t.Fatal("expected error for memory store outside testing, got nil")
			}
		})
	}
}

func TestValidationRules_Memory_AllowedInTesting(t *testing.T) {
	v := validator.New()
	ValidationRules(StoreTypeMemory, "testing", "", nil)(v)
	err := v.Struct(Settings{CookieName: "session"})
	if err != nil {
		t.Fatalf("expected no error for memory store in testing, got %v", err)
	}
}

func TestValidationRules_Cookie_RequiresKey(t *testing.T) {
	v := validator.New()
	ValidationRules(StoreTypeCookie, "production", "", nil)(v) // nil / short cookieKey
	err := v.Struct(Settings{CookieName: "session"})
	if err == nil {
		t.Fatal("expected error for cookie store without key, got nil")
	}
}
