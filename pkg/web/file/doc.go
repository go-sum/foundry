// Package file provides RFC 7232/7233 conformant HTTP file serving.
// All conditional request headers (If-Match, If-Unmodified-Since,
// If-None-Match, If-Modified-Since, If-Range, Range) are handled in
// the correct precedence order. File sources are accessed via the Source
// interface, with os.Root-backed implementations that make path traversal
// structurally impossible.
package file
