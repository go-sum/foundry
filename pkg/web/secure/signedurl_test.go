package secure

import (
	"errors"
	"testing"
	"time"
)

func TestSignURL_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}

	original := "/downloads/report.pdf"
	signed, err := SignURL(original, key, time.Hour)
	if err != nil {
		t.Fatalf("SignURL: %v", err)
	}

	got, err := VerifyURL(signed, key)
	if err != nil {
		t.Fatalf("VerifyURL: %v", err)
	}
	if got != original {
		t.Errorf("VerifyURL returned %q, want %q", got, original)
	}
}

func TestSignURL_PathWithExistingQueryParams(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}

	original := "/files/report.pdf?format=pdf"
	signed, err := SignURL(original, key, time.Hour)
	if err != nil {
		t.Fatalf("SignURL: %v", err)
	}

	// Signature must be appended with & not ?.
	if len(signed) <= len(original) {
		t.Fatalf("signed URL %q is not longer than original %q", signed, original)
	}

	got, err := VerifyURL(signed, key)
	if err != nil {
		t.Fatalf("VerifyURL: %v", err)
	}
	if got != original {
		t.Errorf("VerifyURL returned %q, want %q", got, original)
	}
}

func TestSignURL_Expired(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}

	signed, err := SignURL("/resource", key, time.Millisecond)
	if err != nil {
		t.Fatalf("SignURL: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	_, err = VerifyURL(signed, key)
	if !errors.Is(err, ErrTokenExpired) {
		t.Errorf("VerifyURL(expired) = %v, want %v", err, ErrTokenExpired)
	}
}

func TestSignURL_TamperedPath(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}

	signed, err := SignURL("/downloads/report.pdf", key, time.Hour)
	if err != nil {
		t.Fatalf("SignURL: %v", err)
	}

	// Change the path component.
	tampered := "/downloads/other.pdf" + signed[len("/downloads/report.pdf"):]

	_, err = VerifyURL(tampered, key)
	if !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("VerifyURL(tampered path) = %v, want %v", err, ErrTokenInvalid)
	}
}

func TestSignURL_TamperedQueryParam(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}

	original := "/files/report.pdf?format=pdf"
	signed, err := SignURL(original, key, time.Hour)
	if err != nil {
		t.Fatalf("SignURL: %v", err)
	}

	// Swap format value.
	tampered := signed[:len("/files/report.pdf?format=")] + "csv" + signed[len("/files/report.pdf?format=pdf"):]

	_, err = VerifyURL(tampered, key)
	if !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("VerifyURL(tampered query) = %v, want %v", err, ErrTokenInvalid)
	}
}

func TestVerifyURL_MissingSigParam(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}

	_, err := VerifyURL("/downloads/report.pdf", key)
	if !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("VerifyURL(no sig) = %v, want %v", err, ErrTokenInvalid)
	}
}

func TestSignURL_WrongKey(t *testing.T) {
	keyA := make([]byte, 32)
	for i := range keyA {
		keyA[i] = byte(i + 1)
	}
	keyB := make([]byte, 32)
	for i := range keyB {
		keyB[i] = byte(i + 100)
	}

	signed, err := SignURL("/resource", keyA, time.Hour)
	if err != nil {
		t.Fatalf("SignURL: %v", err)
	}

	_, err = VerifyURL(signed, keyB)
	if !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("VerifyURL(wrong key) = %v, want %v", err, ErrTokenInvalid)
	}
}

func TestSignURL_ShortKey(t *testing.T) {
	shortKey := make([]byte, 16) // under 32 bytes

	_, err := SignURL("/resource", shortKey, time.Hour)
	if err == nil {
		t.Error("SignURL with short key should return an error, got nil")
	}
}
