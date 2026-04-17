package formdata

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

func BenchmarkParse_URLEncoded(b *testing.B) {
	body := "name=alice&email=alice%40example.com&age=30&city=New+York"
	b.ResetTimer()
	for range b.N {
		r := strings.NewReader(body)
		fd, err := Parse(r, "application/x-www-form-urlencoded", DefaultOptions)
		if err != nil {
			b.Fatal(err)
		}
		fd.Close()
	}
}

func BenchmarkParse_Multipart_SmallFields(b *testing.B) {
	var buf bytes.Buffer
	boundary := "benchboundary"
	ct := "multipart/form-data; boundary=" + boundary
	for _, kv := range [][2]string{{"name", "alice"}, {"age", "30"}, {"city", "NYC"}} {
		fmt.Fprintf(&buf, "--%s\r\nContent-Disposition: form-data; name=%q\r\n\r\n%s\r\n", boundary, kv[0], kv[1])
	}
	fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	body := buf.Bytes()

	b.ResetTimer()
	for range b.N {
		fd, err := Parse(bytes.NewReader(body), ct, DefaultOptions)
		if err != nil {
			b.Fatal(err)
		}
		fd.Close()
	}
}

func BenchmarkParse_Multipart_1MBFile(b *testing.B) {
	boundary := "benchfileboundary"
	ct := "multipart/form-data; boundary=" + boundary
	fileContent := bytes.Repeat([]byte("X"), 1<<20) // 1 MiB

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "--%s\r\nContent-Disposition: form-data; name=\"file\"; filename=\"big.bin\"\r\nContent-Type: application/octet-stream\r\n\r\n", boundary)
	buf.Write(fileContent)
	fmt.Fprintf(&buf, "\r\n--%s--\r\n", boundary)
	body := buf.Bytes()

	opts := DefaultOptions
	opts.MaxFileSize = 2 << 20
	opts.MaxTotalSize = 2 << 20

	b.SetBytes(int64(len(fileContent)))
	b.ResetTimer()
	for range b.N {
		fd, err := Parse(bytes.NewReader(body), ct, opts)
		if err != nil {
			b.Fatal(err)
		}
		// Drain the file to simulate reading.
		if len(fd.Files) > 0 {
			for _, files := range fd.Files {
				for _, f := range files {
					rc, _ := f.Open()
					if rc != nil {
						_, _ = io.Copy(io.Discard, rc)
						rc.Close()
					}
				}
			}
		}
		fd.Close()
	}
}
