package auth

import (
	"errors"
	"testing"
	"time"
)

var testMasterKey = make([]byte, 32) // all-zero 32-byte key

func TestDeriveTokenSubkeys_ProducesDistinctKeys(t *testing.T) {
	vk, ik, err := DeriveTokenSubkeys([][]byte{testMasterKey})
	if err != nil {
		t.Fatalf("DeriveTokenSubkeys() error = %v", err)
	}
	if len(vk) != 1 || len(ik) != 1 {
		t.Fatalf("expected 1 verify key and 1 identity key, got %d and %d", len(vk), len(ik))
	}
	if len(vk[0]) != 32 {
		t.Errorf("verify key length = %d, want 32", len(vk[0]))
	}
	if len(ik[0]) != 32 {
		t.Errorf("identity key length = %d, want 32", len(ik[0]))
	}
	if string(vk[0]) == string(ik[0]) {
		t.Error("verify key and identity key are identical — domain separation failed")
	}
}

func TestDeriveTokenSubkeys_Deterministic(t *testing.T) {
	vk1, ik1, _ := DeriveTokenSubkeys([][]byte{testMasterKey})
	vk2, ik2, _ := DeriveTokenSubkeys([][]byte{testMasterKey})
	if string(vk1[0]) != string(vk2[0]) {
		t.Error("verify key is not deterministic")
	}
	if string(ik1[0]) != string(ik2[0]) {
		t.Error("identity key is not deterministic")
	}
}

func TestDeriveTokenSubkeys_EmptyInput_ReturnsEmptySlices(t *testing.T) {
	vk, ik, err := DeriveTokenSubkeys(nil)
	if err != nil {
		t.Fatalf("DeriveTokenSubkeys(nil) error = %v", err)
	}
	if len(vk) != 0 || len(ik) != 0 {
		t.Errorf("expected empty slices, got verifyKeys=%d identityKeys=%d", len(vk), len(ik))
	}
}

// TestDeriveTokenSubkeys_CrossCodecIsolation verifies that a token encoded by
// the verify codec cannot be decoded by the identity codec. This is the
// property that makes key-compromise of one system irrelevant to the other.
func TestDeriveTokenSubkeys_CrossCodecIsolation(t *testing.T) {
	verifyKeys, identityKeys, err := DeriveTokenSubkeys([][]byte{testMasterKey})
	if err != nil {
		t.Fatalf("DeriveTokenSubkeys() error = %v", err)
	}

	verifyCodec, err := NewTokenCodec(verifyKeys)
	if err != nil {
		t.Fatalf("NewTokenCodec(verifyKeys) error = %v", err)
	}
	identityCodec, err := NewTokenCodec(identityKeys)
	if err != nil {
		t.Fatalf("NewTokenCodec(identityKeys) error = %v", err)
	}

	token := VerificationToken{
		Purpose:   FlowSignin,
		Email:     "test@example.com",
		Secret:    "secret",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	encoded, err := verifyCodec.Encode(token)
	if err != nil {
		t.Fatalf("verifyCodec.Encode() error = %v", err)
	}

	_, err = identityCodec.Decode(encoded)
	if err == nil {
		t.Fatal("identityCodec.Decode() succeeded on a verify-codec token — domain separation broken")
	}
	if !errors.Is(err, ErrVerificationMissing) {
		t.Errorf("identityCodec.Decode() error = %v, want ErrVerificationMissing", err)
	}
}
