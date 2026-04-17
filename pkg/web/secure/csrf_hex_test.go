package secure

import (
	"strings"
	"testing"
	"time"
)

func TestNewCSRFConfigFromHex(t *testing.T) {
	validHex := strings.Repeat("ab", 32)   // 64 hex chars = 32 bytes
	altHex := strings.Repeat("cd", 32)     // 64 hex chars = 32 bytes, different value
	shortHex := strings.Repeat("ab", 31)   // 62 hex chars = 31 bytes (too short)

	tests := []struct {
		name            string
		keyHex          string
		previousKeysHex string
		wantErr         bool
		errContains     string
		wantKeyLen      int
		wantKeyNil      bool
		wantPrevCount   int
		wantPrevNil     bool
		checkDefaults   bool
	}{
		{
			name:            "both empty — no error, Key nil, PreviousKeys nil",
			keyHex:          "",
			previousKeysHex: "",
			wantErr:         false,
			wantKeyNil:      true,
			wantPrevNil:     true,
		},
		{
			name:            "valid key only — no error, len(Key)==32, PreviousKeys nil",
			keyHex:          validHex,
			previousKeysHex: "",
			wantErr:         false,
			wantKeyLen:      32,
			wantPrevNil:     true,
		},
		{
			name:            "valid key + one previous — no error, len(Key)==32, len(PreviousKeys)==1",
			keyHex:          validHex,
			previousKeysHex: validHex,
			wantErr:         false,
			wantKeyLen:      32,
			wantPrevCount:   1,
		},
		{
			name:            "valid key + two previous CSV — no error, len(PreviousKeys)==2",
			keyHex:          validHex,
			previousKeysHex: validHex + "," + altHex,
			wantErr:         false,
			wantKeyLen:      32,
			wantPrevCount:   2,
		},
		{
			name:            "invalid key hex — error contains 'csrf key'",
			keyHex:          "not-hex",
			previousKeysHex: "",
			wantErr:         true,
			errContains:     "csrf key",
		},
		{
			name:            "key too short (31 bytes) — error contains 'csrf key'",
			keyHex:          shortHex,
			previousKeysHex: "",
			wantErr:         true,
			errContains:     "csrf key",
		},
		{
			name:            "invalid previous hex — error contains 'csrf previous keys'",
			keyHex:          validHex,
			previousKeysHex: "not-hex",
			wantErr:         true,
			errContains:     "csrf previous keys",
		},
		{
			name:            "defaults preserved — TokenTTL==time.Hour, ContextKey==\"csrf\"",
			keyHex:          validHex,
			previousKeysHex: "",
			wantErr:         false,
			wantKeyLen:      32,
			wantPrevNil:     true,
			checkDefaults:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewCSRFConfigFromHex(tt.keyHex, tt.previousKeysHex)

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

			if tt.wantPrevNil {
				if cfg.PreviousKeys != nil {
					t.Errorf("PreviousKeys = %v, want nil", cfg.PreviousKeys)
				}
			} else {
				if len(cfg.PreviousKeys) != tt.wantPrevCount {
					t.Errorf("len(PreviousKeys) = %d, want %d", len(cfg.PreviousKeys), tt.wantPrevCount)
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
