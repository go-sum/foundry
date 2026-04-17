package secure

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestDecodeHexKey_Valid(t *testing.T) {
	input := strings.Repeat("ab", 32) // 64 hex chars = 32 bytes
	key, err := decodeHexKey(input)
	if err != nil {
		t.Fatalf("decodeHexKey() error = %v", err)
	}
	if len(key) != 32 {
		t.Errorf("len(key) = %d, want 32", len(key))
	}
}

func TestDecodeHexKey_TooShort(t *testing.T) {
	input := strings.Repeat("ab", 31) // 62 hex chars = 31 bytes
	_, err := decodeHexKey(input)
	if err == nil {
		t.Fatal("expected error for 31-byte key, got nil")
	}
}

func TestDecodeHexKey_NotHex(t *testing.T) {
	_, err := decodeHexKey("not-hex!!")
	if err == nil {
		t.Fatal("expected error for non-hex input, got nil")
	}
}

func TestDecodeHexKey_TrimsWhitespace(t *testing.T) {
	input := strings.Repeat("ab", 32) + "\n"
	key, err := decodeHexKey(input)
	if err != nil {
		t.Fatalf("decodeHexKey() error = %v; trailing newline should be trimmed", err)
	}
	if len(key) != 32 {
		t.Errorf("len(key) = %d, want 32", len(key))
	}
}

func TestDecodeHexKeys_MultiValue(t *testing.T) {
	hexA := strings.Repeat("aa", 32)
	hexB := strings.Repeat("bb", 32)
	keys, err := decodeHexKeys(hexA + "," + hexB)
	if err != nil {
		t.Fatalf("decodeHexKeys() error = %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("len(keys) = %d, want 2", len(keys))
	}
}

func TestDecodeHexKeys_SkipEmpty(t *testing.T) {
	hexA := strings.Repeat("aa", 32)
	hexB := strings.Repeat("bb", 32)
	keys, err := decodeHexKeys(hexA + ",," + hexB)
	if err != nil {
		t.Fatalf("decodeHexKeys() error = %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("len(keys) = %d, want 2 (empty entry skipped)", len(keys))
	}
}

func TestDecodeHexKeys_InvalidEntry(t *testing.T) {
	hexA := strings.Repeat("aa", 32)
	_, err := decodeHexKeys(hexA + ",not-hex")
	if err == nil {
		t.Fatal("expected error when one entry is invalid, got nil")
	}
}

func TestGenerateKeyHex_Shape(t *testing.T) {
	s, err := GenerateKeyHex()
	if err != nil {
		t.Fatalf("GenerateKeyHex() error = %v", err)
	}
	if len(s) != 64 {
		t.Errorf("len(hex) = %d, want 64", len(s))
	}
	raw, err := hex.DecodeString(s)
	if err != nil {
		t.Errorf("GenerateKeyHex() output is not valid hex: %v", err)
	}
	if len(raw) != 32 {
		t.Errorf("decoded len = %d, want 32", len(raw))
	}
}
