package cookiecodec_test

import (
	"testing"
	"time"

	"github.com/go-sum/web/cookiecodec"
)

func FuzzParseSigned(f *testing.F) {
	c, _ := cookiecodec.New(cookiecodec.Config{Name: "sess", Secrets: [][]byte{[]byte("secret1")}, Mode: cookiecodec.Signed})
	// Add valid seed
	enc, _ := c.Serialize("hello", time.Time{})
	f.Add(enc)
	f.Add("")
	f.Add("not-base64!!!")
	f.Add("AgAAAAA=") // too short
	f.Fuzz(func(t *testing.T, input string) {
		c2, _ := cookiecodec.New(cookiecodec.Config{Name: "sess", Secrets: [][]byte{[]byte("secret1")}, Mode: cookiecodec.Signed})
		_, _ = c2.Parse(input)
		// Must not panic
	})
}

func FuzzParseAEAD(f *testing.F) {
	c, _ := cookiecodec.New(cookiecodec.Config{Name: "sess", Secrets: [][]byte{[]byte("secret1")}, Mode: cookiecodec.AEAD})
	enc, _ := c.Serialize("hello", time.Time{})
	f.Add(enc)
	f.Add("")
	f.Add("not-base64")
	f.Fuzz(func(t *testing.T, input string) {
		c2, _ := cookiecodec.New(cookiecodec.Config{Name: "sess", Secrets: [][]byte{[]byte("secret1")}, Mode: cookiecodec.AEAD})
		_, _ = c2.Parse(input)
		// Must not panic
	})
}
