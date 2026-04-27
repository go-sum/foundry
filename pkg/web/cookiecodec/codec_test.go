package cookiecodec_test

import (
	"errors"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/web/cookiecodec"
)

func mustNew(t *testing.T, cfg cookiecodec.Config) *cookiecodec.Codec {
	t.Helper()
	c, err := cookiecodec.New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return c
}

// TestP0_01_CookieCodec_NameBinding verifies that a value signed for one
// cookie name cannot be parsed with a different name.
func TestP0_01_CookieCodec_NameBinding(t *testing.T) {
	for _, mode := range []cookiecodec.Mode{cookiecodec.Signed, cookiecodec.AEAD} {
		signer := mustNew(t, cookiecodec.Config{
			Name:    "session",
			Secrets: [][]byte{[]byte("secretkey")},
			Mode:    mode,
		})
		attacker := mustNew(t, cookiecodec.Config{
			Name:    "admin",
			Secrets: [][]byte{[]byte("secretkey")},
			Mode:    mode,
		})

		encoded, err := signer.Serialize("sensitive-value", time.Time{})
		if err != nil {
			t.Fatalf("mode=%d Serialize() error = %v", mode, err)
		}

		_, err = attacker.Parse(encoded)
		if err == nil {
			t.Errorf("mode=%d Parse() with wrong name succeeded, want error", mode)
		}
	}
}

// TestP0_15_CookieCodec_EmptySecretsRejected verifies that New rejects empty secrets.
func TestP0_15_CookieCodec_EmptySecretsRejected(t *testing.T) {
	_, err := cookiecodec.New(cookiecodec.Config{Name: "x", Secrets: nil})
	if !errors.Is(err, cookiecodec.ErrEmptySecrets) {
		t.Fatalf("New() with nil secrets want ErrEmptySecrets, got %v", err)
	}

	_, err = cookiecodec.New(cookiecodec.Config{Name: "x", Secrets: [][]byte{{}}})
	if err == nil {
		t.Fatal("New() with empty secret byte slice should return error")
	}

	_, err = cookiecodec.New(cookiecodec.Config{Name: "", Secrets: [][]byte{[]byte("k")}})
	if err == nil {
		t.Fatal("New() with empty name should return error")
	}
}

// TestCookieCodec_Signed_RoundTrip verifies sign → parse returns the same value.
func TestCookieCodec_Signed_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"simple", "hello"},
		{"json", `{"user":"alice","role":"admin"}`},
		{"empty", ""},
		{"unicode", "日本語テスト"},
	}

	c := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("roundtrip-secret")},
		Mode:    cookiecodec.Signed,
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := c.Serialize(tc.value, time.Time{})
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}
			got, err := c.Parse(encoded)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got != tc.value {
				t.Fatalf("Parse() = %q, want %q", got, tc.value)
			}
		})
	}
}

// TestCookieCodec_Signed_Expiry verifies expiry enforcement.
func TestCookieCodec_Signed_Expiry(t *testing.T) {
	c := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("expiry-secret")},
		Mode:    cookiecodec.Signed,
	})

	// Past expiry should return ErrExpired.
	past := time.Now().Add(-1 * time.Second)
	encoded, err := c.Serialize("value", past)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	_, err = c.Parse(encoded)
	if !errors.Is(err, cookiecodec.ErrExpired) {
		t.Fatalf("Parse() with past exp = %v, want ErrExpired", err)
	}

	// Future expiry should succeed.
	future := time.Now().Add(1 * time.Hour)
	encoded2, err := c.Serialize("value", future)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	got, err := c.Parse(encoded2)
	if err != nil {
		t.Fatalf("Parse() with future exp error = %v", err)
	}
	if got != "value" {
		t.Fatalf("Parse() = %q, want %q", got, "value")
	}
}

// TestCookieCodec_Signed_Tamper verifies that a modified blob fails verification.
func TestCookieCodec_Signed_Tamper(t *testing.T) {
	c := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("tamper-secret")},
		Mode:    cookiecodec.Signed,
	})

	encoded, err := c.Serialize("original", time.Time{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Flip a byte in the middle of the base64 string.
	b := []byte(encoded)
	mid := len(b) / 2
	b[mid] ^= 0xFF
	tampered := string(b)

	_, err = c.Parse(tampered)
	if err == nil {
		t.Fatal("Parse() tampered value succeeded, want error")
	}
}

// TestCookieCodec_Signed_Rotation verifies key rotation: old key still works
// when listed as a fallback, and fails when removed.
func TestCookieCodec_Signed_Rotation(t *testing.T) {
	secretA := []byte("secret-A")
	secretB := []byte("secret-B")

	signerA := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{secretA},
		Mode:    cookiecodec.Signed,
	})
	encoded, err := signerA.Serialize("payload", time.Time{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// [B, A] — A is fallback; should succeed.
	rotated := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{secretB, secretA},
		Mode:    cookiecodec.Signed,
	})
	got, err := rotated.Parse(encoded)
	if err != nil {
		t.Fatalf("Parse() with [B,A] error = %v", err)
	}
	if got != "payload" {
		t.Fatalf("Parse() = %q, want %q", got, "payload")
	}

	// [B] only — A removed; should fail.
	onlyB := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{secretB},
		Mode:    cookiecodec.Signed,
	})
	_, err = onlyB.Parse(encoded)
	if !errors.Is(err, cookiecodec.ErrInvalid) {
		t.Fatalf("Parse() with [B] only = %v, want ErrInvalid", err)
	}
}

// TestCookieCodec_AEAD_RoundTrip verifies encrypt → decrypt returns the same value.
func TestCookieCodec_AEAD_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"simple", "hello"},
		{"json", `{"user":"alice","role":"admin"}`},
		{"empty", ""},
		{"unicode", "日本語テスト"},
	}

	c := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("aead-roundtrip-secret")},
		Mode:    cookiecodec.AEAD,
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := c.Serialize(tc.value, time.Time{})
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}
			got, err := c.Parse(encoded)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got != tc.value {
				t.Fatalf("Parse() = %q, want %q", got, tc.value)
			}
		})
	}
}

// TestCookieCodec_AEAD_NameBinding verifies cross-cookie replay is blocked for AEAD.
func TestCookieCodec_AEAD_NameBinding(t *testing.T) {
	// Already covered by TestP0_01_CookieCodec_NameBinding but kept explicit.
	signer := mustNew(t, cookiecodec.Config{
		Name:    "session",
		Secrets: [][]byte{[]byte("aead-name-secret")},
		Mode:    cookiecodec.AEAD,
	})
	attacker := mustNew(t, cookiecodec.Config{
		Name:    "admin",
		Secrets: [][]byte{[]byte("aead-name-secret")},
		Mode:    cookiecodec.AEAD,
	})

	encoded, err := signer.Serialize("payload", time.Time{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	_, err = attacker.Parse(encoded)
	if err == nil {
		t.Fatal("AEAD Parse() with wrong name succeeded, want error")
	}
}

// TestCookieCodec_AEAD_Tamper verifies that a modified ciphertext byte fails Poly1305.
func TestCookieCodec_AEAD_Tamper(t *testing.T) {
	c := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{[]byte("aead-tamper-secret")},
		Mode:    cookiecodec.AEAD,
	})

	encoded, err := c.Serialize("original", time.Time{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	b := []byte(encoded)
	mid := len(b) / 2
	b[mid] ^= 0x01
	tampered := string(b)

	_, err = c.Parse(tampered)
	if err == nil {
		t.Fatal("AEAD Parse() tampered ciphertext succeeded, want error")
	}
}

// TestCookieCodec_AEAD_Rotation verifies key rotation for AEAD mode.
func TestCookieCodec_AEAD_Rotation(t *testing.T) {
	secretA := []byte("aead-secret-A")
	secretB := []byte("aead-secret-B")

	signerA := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{secretA},
		Mode:    cookiecodec.AEAD,
	})
	encoded, err := signerA.Serialize("payload", time.Time{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// [B, A] — A is fallback; should succeed.
	rotated := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{secretB, secretA},
		Mode:    cookiecodec.AEAD,
	})
	got, err := rotated.Parse(encoded)
	if err != nil {
		t.Fatalf("AEAD Parse() with [B,A] error = %v", err)
	}
	if got != "payload" {
		t.Fatalf("AEAD Parse() = %q, want %q", got, "payload")
	}

	// [B] only — A removed; should fail.
	onlyB := mustNew(t, cookiecodec.Config{
		Name:    "sess",
		Secrets: [][]byte{secretB},
		Mode:    cookiecodec.AEAD,
	})
	_, err = onlyB.Parse(encoded)
	if !errors.Is(err, cookiecodec.ErrInvalid) {
		t.Fatalf("AEAD Parse() with [B] only = %v, want ErrInvalid", err)
	}
}

// TestCookieCodec_SessionCookie verifies that zero exp means no server-side expiry.
func TestCookieCodec_SessionCookie(t *testing.T) {
	for _, mode := range []cookiecodec.Mode{cookiecodec.Signed, cookiecodec.AEAD} {
		c := mustNew(t, cookiecodec.Config{
			Name:    "sess",
			Secrets: [][]byte{[]byte("session-secret")},
			Mode:    mode,
		})

		encoded, err := c.Serialize("no-expiry", time.Time{})
		if err != nil {
			t.Fatalf("mode=%d Serialize() error = %v", mode, err)
		}

		time.Sleep(1 * time.Millisecond)

		got, err := c.Parse(encoded)
		if err != nil {
			t.Fatalf("mode=%d Parse() session cookie error = %v", mode, err)
		}
		if got != "no-expiry" {
			t.Fatalf("mode=%d Parse() = %q, want %q", mode, got, "no-expiry")
		}
	}
}
