package secure

import (
	"strings"
	"testing"
	"time"
)

func TestNewCSRFConfigFromHex(t *testing.T) {
	validHex := strings.Repeat("ab", 32) // 64 hex chars = 32 bytes
	shortHex := strings.Repeat("ab", 31) // 62 hex chars = 31 bytes (too short)

	tests := []struct {
		name          string
		keyHex        string
		wantErr       bool
		errContains   string
		wantKeyLen    int
		wantKeyNil    bool
		checkDefaults bool
	}{
		{
			name:       "empty key — no error, Key nil",
			keyHex:     "",
			wantErr:    false,
			wantKeyNil: true,
		},
		{
			name:       "valid key — no error, len(Key)==32",
			keyHex:     validHex,
			wantErr:    false,
			wantKeyLen: 32,
		},
		{
			name:        "invalid key hex — error contains 'csrf key'",
			keyHex:      "not-hex",
			wantErr:     true,
			errContains: "csrf key",
		},
		{
			name:        "key too short (31 bytes) — error contains 'csrf key'",
			keyHex:      shortHex,
			wantErr:     true,
			errContains: "csrf key",
		},
		{
			name:          "defaults preserved — TokenTTL==time.Hour, ContextKey==\"csrf\"",
			keyHex:        validHex,
			wantErr:       false,
			wantKeyLen:    32,
			checkDefaults: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewCSRFConfigFromHex(tt.keyHex)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("NewCSRFConfigFromHex() error = nil, want error containing %q", tt.errContains)
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewCSRFConfigFromHex() unexpected error = %v", err)
			}

			if tt.wantKeyNil {
				if cfg.Key != nil {
					t.Errorf("Key = %v, want nil", cfg.Key)
				}
			} else {
				if len(cfg.Key) != tt.wantKeyLen {
					t.Errorf("len(Key) = %d, want %d", len(cfg.Key), tt.wantKeyLen)
				}
			}

			if tt.checkDefaults {
				if cfg.TokenTTL != time.Hour {
					t.Errorf("TokenTTL = %v, want %v", cfg.TokenTTL, time.Hour)
				}
				if cfg.ContextKey != "csrf" {
					t.Errorf("ContextKey = %q, want %q", cfg.ContextKey, "csrf")
				}
			}
		})
	}
}
