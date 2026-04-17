package file

import (
	"io/fs"
	"time"
)

// ETagMode controls ETag generation.
type ETagMode int

const (
	// WeakETag generates W/"<size>-<mtime-unix>" — fast, no I/O.
	WeakETag ETagMode = iota
	// StrongETag generates a SHA-256 hash of the file content — requires reading the file.
	StrongETag
)

// Source represents a readable, seekable content source for file serving.
type Source interface {
	// Size returns the content length in bytes.
	Size() int64
	// ModTime returns the last-modified time.
	ModTime() time.Time
	// ReadAt reads len(p) bytes starting at byte offset off.
	// It returns the number of bytes read and any error encountered.
	ReadAt(p []byte, off int64) (n int, err error)
	// ContentType returns the MIME type of the content, or "" for auto-detection.
	ContentType() string
	// Name returns the file name for MIME detection and Content-Disposition.
	Name() string
}

// FileInfo is a value type returned by stat-like operations on a Source.
type FileInfo struct {
	Size    int64
	ModTime time.Time
	IsDir   bool
}

// FSFileInfo adapts fs.FileInfo to Source metadata.
func FSFileInfo(fi fs.FileInfo) FileInfo {
	return FileInfo{Size: fi.Size(), ModTime: fi.ModTime(), IsDir: fi.IsDir()}
}
