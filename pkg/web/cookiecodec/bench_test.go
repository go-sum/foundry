package cookiecodec

import (
	"testing"
	"time"
)

var benchSecret = []byte("32-byte-benchmark-secret-key-here!!")

func BenchmarkCodec_Sign_Signed(b *testing.B) {
	c, _ := New(Config{Name: "sess", Secrets: [][]byte{benchSecret}, Mode: Signed})
	exp := time.Now().Add(time.Hour)
	b.ResetTimer()
	for range b.N {
		if _, err := c.Serialize("user-id-value-1234", exp); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCodec_Sign_AEAD(b *testing.B) {
	c, _ := New(Config{Name: "sess", Secrets: [][]byte{benchSecret}, Mode: AEAD})
	exp := time.Now().Add(time.Hour)
	b.ResetTimer()
	for range b.N {
		if _, err := c.Serialize("user-id-value-1234", exp); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCodec_Parse_Signed(b *testing.B) {
	c, _ := New(Config{Name: "sess", Secrets: [][]byte{benchSecret}, Mode: Signed})
	encoded, _ := c.Serialize("user-id-value-1234", time.Now().Add(time.Hour))
	b.ResetTimer()
	for range b.N {
		if _, err := c.Parse(encoded); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCodec_Parse_AEAD(b *testing.B) {
	c, _ := New(Config{Name: "sess", Secrets: [][]byte{benchSecret}, Mode: AEAD})
	encoded, _ := c.Serialize("user-id-value-1234", time.Now().Add(time.Hour))
	b.ResetTimer()
	for range b.N {
		if _, err := c.Parse(encoded); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCodec_Rotation_3Secrets(b *testing.B) {
	secrets := [][]byte{
		[]byte("new-secret-32bytes-padding-here!!"),
		[]byte("old-secret-32bytes-padding-here!!"),
		[]byte("older-secret-32bytes-padding-here!"),
	}
	c, _ := New(Config{Name: "sess", Secrets: secrets, Mode: Signed})
	// Encode with the oldest key to exercise full rotation scan.
	cold, _ := New(Config{Name: "sess", Secrets: [][]byte{secrets[2]}, Mode: Signed})
	encoded, _ := cold.Serialize("value", time.Now().Add(time.Hour))
	b.ResetTimer()
	for range b.N {
		if _, err := c.Parse(encoded); err != nil {
			b.Fatal(err)
		}
	}
}
