package auth

import (
	"errors"
	"testing"
)

func TestNewVerifier(t *testing.T) {
	v, err := NewVerifier()
	if err != nil {
		t.Fatalf("NewVerifier error: %v", err)
	}
	if len(v) != 64 {
		t.Fatalf("NewVerifier length = %d, want 64", len(v))
	}
	// Verify only base64url (no padding) characters: A-Z, a-z, 0-9, -, _
	for i, ch := range v {
		if !isBase64URLChar(ch) {
			t.Fatalf("NewVerifier char at index %d = %q is not a base64url character", i, ch)
		}
	}
}

func isBase64URLChar(ch rune) bool {
	return (ch >= 'A' && ch <= 'Z') ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '-' || ch == '_'
}

func TestChallenge_RFC7636Vector(t *testing.T) {
	const verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	const wantChallenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	got, err := Challenge(verifier)
	if err != nil {
		t.Fatalf("Challenge error: %v", err)
	}
	if got != wantChallenge {
		t.Fatalf("Challenge = %q, want %q", got, wantChallenge)
	}
}

func TestChallenge_InvalidVerifier(t *testing.T) {
	cases := []struct {
		name     string
		verifier string
	}{
		{"too short", "short"},
		{"too long", string(make([]byte, 129))},
		{"contains space", "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk "},
		{"contains plus", "dBjftJeZ4CVP+mB92K27uhbUJU1p1r_wW1gFWFOEjXk"},
		{"contains slash", "dBjftJeZ4CVP/mB92K27uhbUJU1p1r_wW1gFWFOEjXk"},
		{"non-ASCII byte", "dBjftJeZ4CVP\x80mB92K27uhbUJU1p1r_wW1gFWFOEj"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Challenge(tc.verifier)
			if !errors.Is(err, ErrInvalidVerifier) {
				t.Fatalf("Challenge(%q) error = %v, want ErrInvalidVerifier", tc.verifier, err)
			}
		})
	}
}

func TestChallenge_DifferentVerifiers(t *testing.T) {
	v1, err := NewVerifier()
	if err != nil {
		t.Fatalf("NewVerifier v1 error: %v", err)
	}
	v2, err := NewVerifier()
	if err != nil {
		t.Fatalf("NewVerifier v2 error: %v", err)
	}

	c1, err := Challenge(v1)
	if err != nil {
		t.Fatalf("Challenge(v1) error: %v", err)
	}
	c2, err := Challenge(v2)
	if err != nil {
		t.Fatalf("Challenge(v2) error: %v", err)
	}

	if c1 == c2 {
		t.Fatal("Challenge produced identical output for two different verifiers")
	}
}
