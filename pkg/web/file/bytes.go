package file

import (
	"cmp"
	"mime"
	"path/filepath"
	"time"
)

// BytesSource is an in-memory Source.
type BytesSource struct {
	data    []byte
	name    string
	modTime time.Time
	ct      string
}

// NewBytesSource creates a BytesSource. If contentType is "", it is inferred
// from the name extension.
func NewBytesSource(name string, data []byte, modTime time.Time, contentType string) *BytesSource {
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(name))
	}
	contentType = cmp.Or(contentType, "application/octet-stream")
	return &BytesSource{data: data, name: name, modTime: modTime, ct: contentType}
}

func (b *BytesSource) Size() int64         { return int64(len(b.data)) }
func (b *BytesSource) ModTime() time.Time  { return b.modTime }
func (b *BytesSource) ContentType() string { return b.ct }
func (b *BytesSource) Name() string        { return b.name }

func (b *BytesSource) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(b.data)) {
		return 0, nil
	}
	n := copy(p, b.data[off:])
	return n, nil
}
