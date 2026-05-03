package secure

import (
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
)

func TestCSRFConfigFromHex(t *testing.T) {
	validHex := strings.Repeat("ab", 32) // 64 hex chars = 32 bytes
	shortHex := strings.Repeat("ab", 31) // 62 hex chars = 31 bytes (too short)

	tests := []struct {
		name       string
		keyHex     string
		wantKeyLen int
	}{
		{
			name:       "empty key — Key remains nil",
			keyHex:     "",
			wantKeyLen: 0,
		},
		{
			name:       "valid key — len(Key)==32",
			keyHex:     validHex,
			wantKeyLen: 32,
		},
		{
			name:       "invalid hex — Key remains nil",
			keyHex:     "not-hex",
			wantKeyLen: 0,
		},
		{
			name:       "key too short (31 bytes) — Key remains nil",
			keyHex:     shortHex,
			wantKeyLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := CSRFConfigFromHex(tt.keyHex)

			if got := len(cfg.Key); got != tt.wantKeyLen {
				t.Errorf("len(Key) = %d, want %d", got, tt.wantKeyLen)
			}
		})
	}
}

func TestCSRFConfigFromHex_DefaultsPreserved(t *testing.T) {
	validHex := strings.Repeat("ab", 32)
	cfg := CSRFConfigFromHex(validHex)

	if cfg.TokenTTL != time.Hour {
		t.Errorf("TokenTTL = %v, want %v", cfg.TokenTTL, time.Hour)
	}
	if cfg.ContextKey != "csrf" {
		t.Errorf("ContextKey = %q, want %q", cfg.ContextKey, "csrf")
	}
	if !cfg.CookieSecure {
		t.Error("CookieSecure = false, want true")
	}
}

func TestCSRFConfig_MissingKey_FailsValidation(t *testing.T) {
	v := validator.New()
	cfg := CSRFConfig{} // Key is nil
	err := v.Struct(cfg)
	if err == nil {
		t.Fatal("expected validation error for nil Key, got nil")
	}
}

func TestCSRFConfig_ShortKey_FailsValidation(t *testing.T) {
	v := validator.New()
	cfg := CSRFConfig{Key: make([]byte, 31)} // one byte short
	err := v.Struct(cfg)
	if err == nil {
		t.Fatal("expected validation error for 31-byte Key, got nil")
	}
}

func TestCSRFConfig_ValidKey_PassesValidation(t *testing.T) {
	v := validator.New()
	cfg := CSRFConfig{Key: make([]byte, 32)}
	err := v.Struct(cfg)
	if err != nil {
		t.Fatalf("expected no error for 32-byte Key, got %v", err)
	}
}
