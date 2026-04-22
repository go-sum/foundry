package formdata

import (
	"io"
	"os"
)

// UploadHandler is a function called for each file part.
// It receives the part headers and a reader for the part body,
// and must return a *LazyFile or an error.
// Returning (nil, nil) discards the upload.
type UploadHandler func(field, filename, contentType string, body io.Reader) (*LazyFile, error)

// DefaultUploadHandler spills files to disk once they exceed memThreshold bytes.
// Files below the threshold are kept in memory.
func DefaultUploadHandler(memThreshold int64) UploadHandler {
	return func(field, filename, contentType string, body io.Reader) (*LazyFile, error) {
		// Read up to memThreshold+1 bytes
		buf := make([]byte, memThreshold+1)
		n, err := io.ReadFull(body, buf)
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			// File fits in memory
			data := make([]byte, n)
			copy(data, buf[:n])
			return newMemLazyFile(field, filename, contentType, data), nil
		}
		if err != nil {
			return nil, err
		}

		// File exceeds threshold — spill to disk
		tmp, err := os.CreateTemp("", "web-upload-*")
		if err != nil {
			return nil, err
		}
		written := int64(n)

		if _, err := tmp.Write(buf); err != nil {
			tmp.Close()           //nolint:errcheck
			os.Remove(tmp.Name()) //nolint:errcheck
			return nil, err
		}

		// Drain the rest of the body to disk
		rest, err := io.Copy(tmp, body)
		if err != nil {
			tmp.Close()           //nolint:errcheck
			os.Remove(tmp.Name()) //nolint:errcheck
			return nil, err
		}
		written += rest

		if _, err := tmp.Seek(0, io.SeekStart); err != nil {
			tmp.Close()           //nolint:errcheck
			os.Remove(tmp.Name()) //nolint:errcheck
			return nil, err
		}

		return newDiskLazyFile(field, filename, contentType, tmp, written), nil
	}
}
