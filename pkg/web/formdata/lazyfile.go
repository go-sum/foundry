package formdata

import (
	"bytes"
	"io"
	"os"
)

// LazyFile is a file upload part that may be backed by memory or disk.
// It implements io.ReadCloser. Call Close to release resources.
type LazyFile struct {
	FieldName   string
	Filename    string
	ContentType string
	Size        int64

	data []byte   // non-nil if in memory
	tmpf *os.File // non-nil if on disk
}

// newMemLazyFile creates an in-memory LazyFile from data.
func newMemLazyFile(field, filename, ct string, data []byte) *LazyFile {
	return &LazyFile{
		FieldName: field, Filename: filename, ContentType: ct,
		Size: int64(len(data)), data: data,
	}
}

// newDiskLazyFile creates a disk-backed LazyFile from a temp file.
// The file cursor must be at the start.
func newDiskLazyFile(field, filename, ct string, f *os.File, size int64) *LazyFile {
	return &LazyFile{
		FieldName: field, Filename: filename, ContentType: ct,
		Size: size, tmpf: f,
	}
}

// Open returns a fresh io.ReadCloser positioned at the start of the file data.
// The caller must Close it when done.
func (lf *LazyFile) Open() (io.ReadCloser, error) {
	if lf.data != nil {
		return io.NopCloser(bytes.NewReader(lf.data)), nil
	}
	if _, err := lf.tmpf.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return io.NopCloser(lf.tmpf), nil
}

// Close releases resources. For disk-backed files, deletes the temp file.
func (lf *LazyFile) Close() error {
	if lf.tmpf != nil {
		name := lf.tmpf.Name()
		lf.tmpf.Close() //nolint:errcheck
		return os.Remove(name)
	}
	return nil
}

// Remove deletes a disk-backed temp file (alias for Close that makes intent clear).
func (lf *LazyFile) Remove() error { return lf.Close() }
