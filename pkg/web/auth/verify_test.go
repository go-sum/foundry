package auth

import (
	"errors"
	"testing"
)

func TestVerifyState_Match(t *testing.T) {
	if err := VerifyState("abc123", "abc123"); err != nil {
		t.Fatalf("VerifyState match error: %v", err)
	}
}

func TestVerifyState_Mismatch(t *testing.T) {
	err := VerifyState("wrong", "abc123")
	if !errors.Is(err, ErrStateMismatch) {
		t.Fatalf("VerifyState mismatch: got %v, want ErrStateMismatch", err)
	}
}

func TestVerifyState_EmptyReturnedNonEmptyExpected(t *testing.T) {
	err := VerifyState("", "abc123")
	if !errors.Is(err, ErrStateMismatch) {
		t.Fatalf("VerifyState empty returned: got %v, want ErrStateMismatch", err)
	}
}

func TestVerifyNonce_Match(t *testing.T) {
	if err := VerifyNonce("nonce42", "nonce42"); err != nil {
		t.Fatalf("VerifyNonce match error: %v", err)
	}
}

func TestVerifyNonce_EmptyExpectedSkipsCheck(t *testing.T) {
	// Any returned value should pass when expected is empty.
	if err := VerifyNonce("anything", ""); err != nil {
		t.Fatalf("VerifyNonce empty expected: got %v, want nil", err)
	}
}

func TestVerifyNonce_Mismatch(t *testing.T) {
	err := VerifyNonce("bad", "nonce42")
	if !errors.Is(err, ErrNonceMismatch) {
		t.Fatalf("VerifyNonce mismatch: got %v, want ErrNonceMismatch", err)
	}
}
