package provider

import (
	"strings"
	"testing"
)

func TestHashToken_Returns64CharHex(t *testing.T) {
	got := HashToken("some-token-value")
	if len(got) != 64 {
		t.Errorf("HashToken length = %d, want 64", len(got))
	}
	for i, ch := range got {
		if !isHexChar(ch) {
			t.Errorf("HashToken char at index %d = %q is not a hex character", i, ch)
		}
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	const input = "deterministic-test-token"
	first := HashToken(input)
	second := HashToken(input)
	if first != second {
		t.Errorf("HashToken not deterministic: first=%q second=%q", first, second)
	}
}

func TestHashToken_DifferentInputsDifferentOutputs(t *testing.T) {
	cases := []struct {
		a string
		b string
	}{
		{"token-a", "token-b"},
		{"", "x"},
		{"abc", "ABC"},
		{"same-length-1", "same-length-2"},
	}
	for _, tc := range cases {
		hashA := HashToken(tc.a)
		hashB := HashToken(tc.b)
		if hashA == hashB {
			t.Errorf("HashToken(%q) == HashToken(%q) = %q; want different values", tc.a, tc.b, hashA)
		}
	}
}

func TestHashToken_EmptyInputProducesKnownHash(t *testing.T) {
	// SHA-256 of empty string is a well-known constant.
	const wantEmptyHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	got := HashToken("")
	if got != wantEmptyHash {
		t.Errorf("HashToken(\"\") = %q, want %q", got, wantEmptyHash)
	}
}

func TestGenerateToken_NonEmpty(t *testing.T) {
	tok, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken error: %v", err)
	}
	if tok == "" {
		t.Fatal("generateToken returned empty string")
	}
}

func TestGenerateToken_TwoCallsDifferent(t *testing.T) {
	t1, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken call 1 error: %v", err)
	}
	t2, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken call 2 error: %v", err)
	}
	if t1 == t2 {
		t.Errorf("generateToken returned identical values on two calls: %q", t1)
	}
}

func TestGenerateToken_Length(t *testing.T) {
	// 32 random bytes base64url-encoded (no padding) = 43 characters.
	tok, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken error: %v", err)
	}
	if len(tok) != 43 {
		t.Errorf("generateToken length = %d, want 43", len(tok))
	}
}

func TestGenerateToken_OnlyBase64URLChars(t *testing.T) {
	tok, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken error: %v", err)
	}
	for i, ch := range tok {
		if !isBase64URLChar(ch) {
			t.Errorf("generateToken char at index %d = %q is not a base64url character", i, ch)
		}
	}
}

// isHexChar reports whether ch is a valid lowercase hex digit.
func isHexChar(ch rune) bool {
	return strings.ContainsRune("0123456789abcdef", ch)
}

// isBase64URLChar reports whether ch is a valid base64url (no padding) character.
func isBase64URLChar(ch rune) bool {
	return (ch >= 'A' && ch <= 'Z') ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '-' || ch == '_'
}
