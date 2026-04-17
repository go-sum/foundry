package file

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// WeakETagFor returns the weak ETag for a source: W/"<size>-<mtime-unix>".
func WeakETagFor(src Source) string {
	return fmt.Sprintf(`W/"%d-%d"`, src.Size(), src.ModTime().Unix())
}

// StrongETagFor computes the SHA-256 hash of the source content and returns
// a strong ETag. This requires reading the entire file.
func StrongETagFor(src Source) (string, error) {
	h := sha256.New()
	size := src.Size()
	buf := make([]byte, 32*1024)
	var off int64
	for off < size {
		n, err := src.ReadAt(buf, off)
		if n > 0 {
			h.Write(buf[:n])
			off += int64(n)
		}
		if err == io.EOF || (err == nil && n == 0) {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return `"` + hex.EncodeToString(h.Sum(nil)) + `"`, nil
}
