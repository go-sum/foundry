package formdata

import (
	"bytes"
	"cmp"
	"fmt"
	"strings"
	"testing"
)

// FuzzParseMultipart feeds arbitrary bytes as a multipart body to Parse.
// The invariants are:
//  1. Parse never panics.
//  2. If Parse returns a non-nil *FormData, calling Close() on it never panics.
//  3. Any resource allocated on error is cleaned up (no goroutine leaks — not
//     directly testable here, but tested by -race + leak-detector in CI).
func FuzzParseMultipart(f *testing.F) {
	// Seed: valid two-field form
	f.Add(
		"boundary123",
		"--boundary123\r\nContent-Disposition: form-data; name=\"field1\"\r\n\r\nvalue1\r\n--boundary123--\r\n",
	)
	// Seed: file upload
	f.Add(
		"bnd",
		"--bnd\r\nContent-Disposition: form-data; name=\"f\"; filename=\"a.txt\"\r\nContent-Type: text/plain\r\n\r\nhello\r\n--bnd--\r\n",
	)
	// Seed: empty body
	f.Add("b", "")
	// Seed: only epilogue
	f.Add("b", "--b--\r\n")
	// Seed: missing final boundary
	f.Add("abc", "--abc\r\nContent-Disposition: form-data; name=\"x\"\r\n\r\nval")
	// Seed: malformed header
	f.Add("b", "--b\r\nBad Header\r\n\r\ndata\r\n--b--\r\n")
	// Seed: boundary appears inside value
	f.Add("B", "--B\r\nContent-Disposition: form-data; name=\"n\"\r\n\r\n--B inside\r\n--B--\r\n")
	// Seed: many small fields (exercises MaxParts)
	{
		var sb strings.Builder
		for i := range 5 {
			fmt.Fprintf(&sb, "--bx\r\nContent-Disposition: form-data; name=\"k%d\"\r\n\r\nv%d\r\n", i, i)
		}
		sb.WriteString("--bx--\r\n")
		f.Add("bx", sb.String())
	}

	f.Fuzz(func(t *testing.T, boundary, rawBody string) {
		boundary = cmp.Or(boundary, "fuzz")
		ct := "multipart/form-data; boundary=" + boundary

		opts := ParseOptions{
			MaxMemory:    64 << 10,  // 64 KiB spill threshold
			MaxFiles:     5,
			MaxFileSize:  128 << 10, // 128 KiB per file
			MaxParts:     20,
			MaxTotalSize: 256 << 10, // 256 KiB total
		}

		fd, err := Parse(bytes.NewBufferString(rawBody), ct, opts)
		if err != nil {
			// Error is fine — must not panic, and fd must be nil.
			if fd != nil {
				t.Error("expected nil FormData on error")
			}
			return
		}
		if fd == nil {
			t.Error("expected non-nil FormData on success")
			return
		}
		fd.Close()
	})
}

// FuzzParseURLEncoded feeds arbitrary strings as URL-encoded form bodies.
func FuzzParseURLEncoded(f *testing.F) {
	f.Add("a=1&b=2")
	f.Add("key=value+with+spaces")
	f.Add("a%3D=%26")
	f.Add("")
	f.Add("x=\x00\r\n")
	f.Add(strings.Repeat("k=v&", 200))

	f.Fuzz(func(t *testing.T, rawBody string) {
		opts := ParseOptions{
			MaxTotalSize: 32 << 10,
		}
		fd, err := Parse(bytes.NewBufferString(rawBody), "application/x-www-form-urlencoded", opts)
		if err != nil {
			if fd != nil {
				t.Error("expected nil FormData on error")
			}
			return
		}
		if fd == nil {
			t.Error("expected non-nil FormData on success")
			return
		}
		fd.Close()
	})
}
