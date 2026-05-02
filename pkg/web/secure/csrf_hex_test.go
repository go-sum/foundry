package secure

import (
	"strings"
	"testing"
	"time"
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
