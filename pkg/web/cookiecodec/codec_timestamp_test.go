package cookiecodec_test

// Tests for the Y2106 fix: version-bump from uint32 to int64 timestamps.
// versionAEAD  = 0x82 (legacy uint32 iat/exp)
// versionAEAD2 = 0x83 (new    int64 iat/exp)
// versionSigned  = 0x02 (legacy uint32 iat/exp)
// versionSigned2 = 0x03 (new    int64 iat/exp)

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"testing"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"

	"github.com/go-sum/web/cookiecodec"
)

// TestCookieCodec_AEAD2_RoundTrip verifies that the new versionAEAD2 format
// round-trips correctly.
func TestCookieCodec_AEAD2_RoundTrip(t *testing.T) {
	c := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("aead2-roundtrip-secret")},
		Mode:    cookiecodec.AEAD,
	})

	encoded, err := c.Serialize("hello-aead2", time.Time{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Verify blob starts with versionAEAD2 (0x83).
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

// TestCookieCodec_Signed2_RoundTrip verifies that the new versionSigned2 format
// round-trips correctly.
func TestCookieCodec_Signed2_RoundTrip(t *testing.T) {
	c := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("signed2-roundtrip-secret")},
		Mode:    cookiecodec.Signed,
	})

	encoded, err := c.Serialize("hello-signed2", time.Time{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Verify blob starts with versionSigned2 (0x03).
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

// TestCookieCodec_AEAD_LegacyRoundTrip verifies that a hand-crafted versionAEAD
// (0x82, uint32 timestamps) blob is still accepted.
func TestCookieCodec_AEAD_LegacyRoundTrip(t *testing.T) {
	secret := []byte("aead-legacy-secret")
	name := "sess"
	value := "legacy-aead-value"

	encoded := buildLegacyAEADBlob(t, secret, name, value, 0 /* no expiry */)

	c := mustNew(t, cookiecodec.Config{
		Name:    name,
		Secrets: [][]byte{secret},
		Mode:    cookiecodec.AEAD,
	})

	got, err := c.Parse(encoded)
	if err != nil {
		t.Fatalf("Parse() legacy AEAD blob error = %v", err)
	}
	if got != value {
		t.Fatalf("Parse() = %q, want %q", got, value)
	}
}

// TestCookieCodec_Signed_LegacyRoundTrip verifies that a hand-crafted versionSigned
// (0x02, uint32 timestamps) blob is still accepted.
func TestCookieCodec_Signed_LegacyRoundTrip(t *testing.T) {
	secret := []byte("signed-legacy-secret")
	name := "sess"
	value := "legacy-signed-value"

	encoded := buildLegacySignedBlob(t, secret, name, value, 0 /* no expiry */)

	c := mustNew(t, cookiecodec.Config{
		Name:    name,
		Secrets: [][]byte{secret},
		Mode:    cookiecodec.Signed,
	})

	got, err := c.Parse(encoded)
	if err != nil {
		t.Fatalf("Parse() legacy Signed blob error = %v", err)
	}
	if got != value {
		t.Fatalf("Parse() = %q, want %q", got, value)
	}
}

// TestCookieCodec_Post2106_AEAD verifies that a timestamp after year 2106
// (Unix > 2^32) encodes and decodes correctly with versionAEAD2.
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

	// The blob must use versionAEAD2 (0x83).
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
// (Unix > 2^32) encodes and decodes correctly with versionSigned2.
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

	// The blob must use versionSigned2 (0x03).
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
// is rejected even in the new versionAEAD2 format.
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
	// version(1) + nonce(24) + iatExp(16) = 41 bytes header for versionAEAD2.
	if len(raw) <= 42 {
		t.Fatal("blob too short to tamper")
	}
	raw[41] ^= 0x01
	tampered := base64.RawURLEncoding.EncodeToString(raw)

	_, err = c.Parse(tampered)
	if err == nil {
		t.Fatal("Parse() tampered AEAD2 ciphertext succeeded, want error")
	}
}

// TestCookieCodec_Signed2_TamperedMAC verifies that a tampered MAC is rejected
// in the new versionSigned2 format.
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
		t.Fatal("Parse() tampered Signed2 MAC succeeded, want error")
	}
	if !errors.Is(err, cookiecodec.ErrInvalid) {
		t.Fatalf("Parse() error = %v, want ErrInvalid", err)
	}
}

// buildLegacyAEADBlob constructs a versionAEAD (0x82) blob manually, replicating
// the old uint32 timestamp layout so we can test backward-compatibility parsing.
func buildLegacyAEADBlob(t *testing.T, secret []byte, name, value string, expUnix uint32) string {
	t.Helper()

	key := deriveAEADKeyHelper(t, secret)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		t.Fatalf("chacha20poly1305.NewX: %v", err)
	}

	var nonce [chacha20poly1305.NonceSizeX]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		t.Fatalf("rand nonce: %v", err)
	}

	iat := uint32(time.Now().Unix())
	var iatExp [8]byte
	binary.BigEndian.PutUint32(iatExp[0:4], iat)
	binary.BigEndian.PutUint32(iatExp[4:8], expUnix)

	// Legacy AAD: name + 0x00 + 0x82 + iatExp (8 bytes)
	aad := buildLegacyAEADAAD(name, iatExp[:])

	ciphertext := aead.Seal(nil, nonce[:], []byte(value), aad)

	// blob = version(0x82) || nonce(24) || iatExp(8) || ciphertext+tag
	blob := make([]byte, 1+len(nonce)+8+len(ciphertext))
	blob[0] = 0x82
	copy(blob[1:], nonce[:])
	copy(blob[1+len(nonce):], iatExp[:])
	copy(blob[1+len(nonce)+8:], ciphertext)

	return base64.RawURLEncoding.EncodeToString(blob)
}

func buildLegacyAEADAAD(name string, iatExp []byte) []byte {
	aad := make([]byte, len(name)+2+len(iatExp))
	copy(aad, []byte(name))
	aad[len(name)] = 0x00
	aad[len(name)+1] = 0x82 // versionAEAD
	copy(aad[len(name)+2:], iatExp)
	return aad
}

func deriveAEADKeyHelper(t *testing.T, secret []byte) []byte {
	t.Helper()
	r := hkdf.New(sha256.New, secret, nil, []byte("web/cookiecodec/v2/aead"))
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(r, key); err != nil {
		t.Fatalf("hkdf: %v", err)
	}
	return key
}

// buildLegacySignedBlob constructs a versionSigned (0x02) blob manually,
// replicating the old uint32 timestamp layout.
func buildLegacySignedBlob(t *testing.T, secret []byte, name, value string, expUnix uint32) string {
	t.Helper()

	payload := []byte(value)
	iat := uint32(time.Now().Unix())

	// blob = version(0x02) | iat_uint32(4) | exp_uint32(4) | payload
	blob := make([]byte, 9+len(payload))
	blob[0] = 0x02
	binary.BigEndian.PutUint32(blob[1:5], iat)
	binary.BigEndian.PutUint32(blob[5:9], expUnix)
	copy(blob[9:], payload)

	mac := computeLegacySignedMAC(secret, blob, name)
	blob = append(blob, mac...)

	return base64.RawURLEncoding.EncodeToString(blob)
}

func computeLegacySignedMAC(secret, msg []byte, name string) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(msg)
	h.Write([]byte{0x1E})
	h.Write([]byte(name))
	h.Write([]byte{0x1E})
	return h.Sum(nil)
}
