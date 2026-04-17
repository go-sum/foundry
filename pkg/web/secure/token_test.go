package secure

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

func TestIssueToken(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	tok, err := IssueToken(key, "csrf", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}
	if tok == "" {
		t.Fatal("IssueToken returned empty string")
	}

	// Must be valid base64url.
	raw, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		t.Fatalf("token is not valid base64url: %v", err)
	}
	if len(raw) != tokenSize {
		t.Fatalf("decoded token length = %d, want %d", len(raw), tokenSize)
	}
}

func TestVerifyToken(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	otherKey := make([]byte, 32)
	for i := range otherKey {
		otherKey[i] = byte(i + 100)
	}

	validToken, err := IssueToken(key, "csrf", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	tests := []struct {
		name    string
		key     []byte
		scope   string
		token   string
		wantErr error
	}{
		{
			name:    "valid token with correct key and scope",
			key:     key,
			scope:   "csrf",
			token:   validToken,
			wantErr: nil,
		},
		{
			name:    "wrong key",
			key:     otherKey,
			scope:   "csrf",
			token:   validToken,
			wantErr: ErrTokenInvalid,
		},
		{
			name:    "wrong scope",
			key:     key,
			scope:   "wrong-scope",
			token:   validToken,
			wantErr: ErrTokenInvalid,
		},
		{
			name:    "garbage input",
			key:     key,
			scope:   "csrf",
			token:   "not-a-valid-token!!!",
			wantErr: ErrTokenInvalid,
		},
		{
			name:    "empty input",
			key:     key,
			scope:   "csrf",
			token:   "",
			wantErr: ErrTokenInvalid,
		},
		{
			name:    "truncated token",
			key:     key,
			scope:   "csrf",
			token:   validToken[:len(validToken)/2],
			wantErr: ErrTokenInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyToken(tt.key, tt.scope, tt.token)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("VerifyToken() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyToken_Tampered(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	tok, err := IssueToken(key, "csrf", time.Hour)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	// Decode, flip a byte in the nonce region, re-encode.
	raw, _ := base64.RawURLEncoding.DecodeString(tok)
	raw[0] ^= 0xFF
	tampered := base64.RawURLEncoding.EncodeToString(raw)

	err = VerifyToken(key, "csrf", tampered)
	if !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("VerifyToken(tampered) = %v, want %v", err, ErrTokenInvalid)
	}
}

func TestVerifyToken_Expired(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	// Issue with 1ms TTL.
	tok, err := IssueToken(key, "csrf", time.Millisecond)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	// Sleep to ensure expiry.
	time.Sleep(10 * time.Millisecond)

	err = VerifyToken(key, "csrf", tok)
	if !errors.Is(err, ErrTokenExpired) {
		t.Errorf("VerifyToken(expired) = %v, want %v", err, ErrTokenExpired)
	}
}
