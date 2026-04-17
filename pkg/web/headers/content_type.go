package headers

import (
	"fmt"
	"strings"
)

// ContentType represents a parsed Content-Type header.
type ContentType struct {
	MediaType string            // e.g. "text/html"
	Charset   string            // from charset parameter (lowercased)
	Boundary  string            // from boundary parameter (for multipart)
	Params    map[string]string // all parameters
}

// ParseContentType parses a Content-Type header value.
func ParseContentType(v string) (ContentType, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return ContentType{}, fmt.Errorf("headers: empty Content-Type")
	}

	var ct ContentType

	parts := strings.Split(v, ";")
	ct.MediaType = strings.ToLower(strings.TrimSpace(parts[0]))

	if len(parts) > 1 {
		paramStr := strings.Join(parts[1:], ";")
		ct.Params = ParseParams(paramStr)
		if ct.Params != nil {
			ct.Charset = strings.ToLower(ct.Params["charset"])
			ct.Boundary = ct.Params["boundary"]
		}
	}

	return ct, nil
}

// String returns the canonical header value.
func (c ContentType) String() string {
	if c.MediaType == "" {
		return ""
	}

	var b strings.Builder
	b.WriteString(c.MediaType)

	// Emit params in a stable order: charset first, boundary second, then rest.
	if c.Charset != "" {
		b.WriteString("; charset=")
		b.WriteString(c.Charset)
	}
	if c.Boundary != "" {
		b.WriteString("; boundary=")
		b.WriteString(Quote(c.Boundary))
	}
	for k, v := range c.Params {
		if k == "charset" || k == "boundary" {
			continue
		}
		b.WriteString("; ")
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(Quote(v))
	}

	return b.String()
}
