package auth

import (
	"errors"
	"testing"
)

func TestParseTokenKeys_Empty_ReturnsErrMissing(t *testing.T) {
	_, err := ParseTokenKeys("")
	if !errors.Is(err, ErrTokenKeyMissing) {
		t.Errorf("error = %v, want ErrTokenKeyMissing", err)
	}
}

func TestParseTokenKeys_Malformed_ReturnsErrInvalid(t *testing.T) {
	_, err := ParseTokenKeys("not-hex")
	if !errors.Is(err, ErrTokenKeyInvalid) {
		t.Errorf("error = %v, want ErrTokenKeyInvalid", err)
	}
}

func TestParseTokenKeys_TooShort_ReturnsErrInvalid(t *testing.T) {
	// 31 bytes = 62 hex chars — one byte short of the 32-byte minimum
	_, err := ParseTokenKeys("0000000000000000000000000000000000000000000000000000000000001")
	if !errors.Is(err, ErrTokenKeyInvalid) {
		t.Errorf("error = %v, want ErrTokenKeyInvalid", err)
	}
}

func TestParseTokenKeys_Valid_ReturnsKey(t *testing.T) {
	keys, err := ParseTokenKeys("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatalf("ParseTokenKeys() error = %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("len(keys) = %d, want 1", len(keys))
	}
	if len(keys[0]) < 32 {
		t.Errorf("len(keys[0]) = %d, want >= 32", len(keys[0]))
	}
}

func TestDeriveTokenSubkeys_ProducesNonEmptyKeys(t *testing.T) {
	masterKeys, err := ParseTokenKeys("0000000000000000000000000000000000000000000000000000000000000002")
	if err != nil {
		t.Fatalf("ParseTokenKeys() error = %v", err)
	}
	verifyKeys, identityKeys, err := DeriveTokenSubkeys(masterKeys)
	if err != nil {
		t.Fatalf("DeriveTokenSubkeys() error = %v", err)
	}
	if len(verifyKeys) == 0 {
		t.Error("verifyKeys is empty, want at least one key")
	}
	if len(verifyKeys[0]) < 32 {
		t.Errorf("len(verifyKeys[0]) = %d, want >= 32", len(verifyKeys[0]))
	}
	if len(identityKeys) == 0 {
		t.Error("identityKeys is empty, want at least one key")
	}
	if len(identityKeys[0]) < 32 {
		t.Errorf("len(identityKeys[0]) = %d, want >= 32", len(identityKeys[0]))
	}
}
