package file

import "strings"

var compressibleMIMETypes = map[string]struct{}{
	"application/json":          {},
	"application/javascript":    {},
	"application/xml":           {},
	"application/xhtml+xml":     {},
	"application/wasm":          {},
	"application/manifest+json": {},
	"image/svg+xml":             {},
}

// isCompressibleMIME reports whether the given MIME type should be compressed.
// Ranges matching are not applied to compressible types to avoid conflicting.
func isCompressibleMIME(ct string) bool {
	if ct == "" {
		return false
	}
	// Strip parameters
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	if strings.HasPrefix(ct, "text/") {
		return true
	}
	if _, ok := compressibleMIMETypes[ct]; ok {
		return true
	}
	// +json, +xml, +text suffixes
	return strings.HasSuffix(ct, "+json") ||
		strings.HasSuffix(ct, "+xml") ||
		strings.HasSuffix(ct, "+text")
}
