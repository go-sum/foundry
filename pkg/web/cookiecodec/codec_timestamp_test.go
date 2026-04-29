package cookiecodec_test

// Tests for the Y2106 fix: version-bump from uint32 to int64 timestamps.
// versionAEAD  = 0x83 (int64 iat/exp)
// versionSigned = 0x03 (int64 iat/exp)

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/web/cookiecodec"
)

// TestCookieCodec_AEAD_VersionByte verifies that the versionAEAD format
// serializes with the correct 0x83 version byte and round-trips correctly.
func TestCookieCodec_AEAD_VersionByte(t *testing.T) {
	c := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("aead2-roundtrip-secret")},
		Mode:    cookiecodec.AEAD,
	})

	encoded, err := c.Serialize("hello-aead2", time.Time{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Verify blob starts with versionAEAD (0x83).
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode error = %v", err)
	}
	if raw[0] != 0x83 {
		t.Fatalf("expected version byte 0x83, got 0x%02x", raw[0])
	}

	got, err := c.Parse(encoded)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got != "hello-aead2" {
		t.Fatalf("Parse() = %q, want %q", got, "hello-aead2")
	}
}

// TestCookieCodec_Signed_VersionByte verifies that the versionSigned format
// serializes with the correct 0x03 version byte and round-trips correctly.
func TestCookieCodec_Signed_VersionByte(t *testing.T) {
	c := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("signed2-roundtrip-secret")},
		Mode:    cookiecodec.Signed,
	})

	encoded, err := c.Serialize("hello-signed2", time.Time{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Verify blob starts with versionSigned (0x03).
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode error = %v", err)
	}
	if raw[0] != 0x03 {
		t.Fatalf("expected version byte 0x03, got 0x%02x", raw[0])
	}

	got, err := c.Parse(encoded)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got != "hello-signed2" {
		t.Fatalf("Parse() = %q, want %q", got, "hello-signed2")
	}
}

// TestCookieCodec_Post2106_AEAD verifies that a timestamp after year 2106
// (Unix > 2^32) encodes and decodes correctly with versionAEAD.
func TestCookieCodec_Post2106_AEAD(t *testing.T) {
	c := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("aead-post2106-secret")},
		Mode:    cookiecodec.AEAD,
	})

	// time.Unix(1<<33, 0) ≈ year 2242, well beyond uint32 overflow.
	future2242 := time.Unix(1<<33, 0)
	encoded, err := c.Serialize("far-future", future2242)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// The blob must use versionAEAD (0x83).
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode error = %v", err)
	}
	if raw[0] != 0x83 {
		t.Fatalf("expected version byte 0x83, got 0x%02x", raw[0])
	}

	got, err := c.Parse(encoded)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got != "far-future" {
		t.Fatalf("Parse() = %q, want %q", got, "far-future")
	}
}

// TestCookieCodec_Post2106_Signed verifies that a timestamp after year 2106
// (Unix > 2^32) encodes and decodes correctly with versionSigned.
func TestCookieCodec_Post2106_Signed(t *testing.T) {
	c := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("signed-post2106-secret")},
		Mode:    cookiecodec.Signed,
	})

	// time.Unix(1<<33, 0) ≈ year 2242.
	future2242 := time.Unix(1<<33, 0)
	encoded, err := c.Serialize("far-future-signed", future2242)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// The blob must use versionSigned (0x03).
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode error = %v", err)
	}
	if raw[0] != 0x03 {
		t.Fatalf("expected version byte 0x03, got 0x%02x", raw[0])
	}

	got, err := c.Parse(encoded)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got != "far-future-signed" {
		t.Fatalf("Parse() = %q, want %q", got, "far-future-signed")
	}
}

// TestCookieCodec_Post2106_Expiry_AEAD verifies that a post-2106 expiry is
// correctly enforced (both unexpired and expired cases).
func TestCookieCodec_Post2106_Expiry_AEAD(t *testing.T) {
	c := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("aead-post2106-exp-secret")},
		Mode:    cookiecodec.AEAD,
	})

	// Future post-2106 expiry: should parse successfully.
	future2242 := time.Unix(1<<33, 0)
	encoded, err := c.Serialize("value", future2242)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	got, err := c.Parse(encoded)
	if err != nil {
		t.Fatalf("Parse() with post-2106 future expiry error = %v", err)
	}
	if got != "value" {
		t.Fatalf("Parse() = %q, want %q", got, "value")
	}
}

// TestCookieCodec_Post2106_Expiry_Signed verifies that a post-2106 expiry is
// correctly enforced for the Signed mode.
func TestCookieCodec_Post2106_Expiry_Signed(t *testing.T) {
	c := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("signed-post2106-exp-secret")},
		Mode:    cookiecodec.Signed,
	})

	future2242 := time.Unix(1<<33, 0)
	encoded, err := c.Serialize("value", future2242)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	got, err := c.Parse(encoded)
	if err != nil {
		t.Fatalf("Parse() with post-2106 future expiry error = %v", err)
	}
	if got != "value" {
		t.Fatalf("Parse() = %q, want %q", got, "value")
	}
}

// TestCookieCodec_AEAD2_TamperedCiphertext verifies that a tampered ciphertext
// is rejected in the versionAEAD format.
func TestCookieCodec_AEAD2_TamperedCiphertext(t *testing.T) {
	c := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("aead2-tamper-secret")},
		Mode:    cookiecodec.AEAD,
	})

	encoded, err := c.Serialize("original", time.Time{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode error = %v", err)
	}

	// Flip a byte in the ciphertext region (past version+nonce+iatExp header).
	// version(1) + nonce(24) + iatExp(16) = 41 bytes header for versionAEAD.
	if len(raw) <= 42 {
		t.Fatal("blob too short to tamper")
	}
	raw[41] ^= 0x01
	tampered := base64.RawURLEncoding.EncodeToString(raw)

	_, err = c.Parse(tampered)
	if err == nil {
		t.Fatal("Parse() tampered AEAD ciphertext succeeded, want error")
	}
}

// TestCookieCodec_Signed2_TamperedMAC verifies that a tampered MAC is rejected
// in the versionSigned format.
func TestCookieCodec_Signed2_TamperedMAC(t *testing.T) {
	c := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("signed2-tamper-secret")},
		Mode:    cookiecodec.Signed,
	})

	encoded, err := c.Serialize("original", time.Time{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode error = %v", err)
	}

	// Flip a byte in the MAC (last sha256.Size bytes).
	raw[len(raw)-1] ^= 0x01
	tampered := base64.RawURLEncoding.EncodeToString(raw)

	_, err = c.Parse(tampered)
	if err == nil {
		t.Fatal("Parse() tampered Signed MAC succeeded, want error")
	}
	if !errors.Is(err, cookiecodec.ErrInvalid) {
		t.Fatalf("Parse() error = %v, want ErrInvalid", err)
	}
}
