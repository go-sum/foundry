package authweb

import (
	"errors"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/auth"
)

var testMasterKey = make([]byte, 32) // all-zero 32-byte key

func TestDeriveTokenSubkeys_CrossCodecIsolation(t *testing.T) {
	verifyKeys, identityKeys, err := auth.DeriveTokenSubkeys([][]byte{testMasterKey})
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

	token := auth.VerificationToken{
		Purpose:   auth.FlowSignin,
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
	if !errors.Is(err, auth.ErrVerificationMissing) {
		t.Errorf("identityCodec.Decode() error = %v, want ErrVerificationMissing", err)
	}
}
