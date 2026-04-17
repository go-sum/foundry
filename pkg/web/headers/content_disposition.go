package headers

import (
	"fmt"
	"net/url"
	"strings"
)

// ContentDisposition represents a parsed Content-Disposition header.
type ContentDisposition struct {
	Type        string // "attachment", "inline", "form-data"
	Name        string // for form-data parts (name parameter)
	Filename    string // filename parameter (raw, may be ASCII only)
	FilenameExt string // filename* parameter decoded per RFC 5987 (UTF-8)
}

// ParseContentDisposition parses a Content-Disposition header value.
// It handles RFC 5987 extended parameters (filename*=UTF-8''...).
func ParseContentDisposition(v string) (ContentDisposition, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return ContentDisposition{}, fmt.Errorf("headers: empty Content-Disposition")
	}

	var cd ContentDisposition

	// Split on semicolon; first part is the disposition type.
	parts := strings.Split(v, ";")
	cd.Type = strings.ToLower(strings.TrimSpace(parts[0]))

	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		idx := strings.IndexByte(part, '=')
		if idx < 0 {
			continue
		}

		k := strings.ToLower(strings.TrimSpace(part[:idx]))
		val := strings.TrimSpace(part[idx+1:])

		switch k {
		case "name":
			cd.Name = decodeParamValue(val)
		case "filename":
			cd.Filename = decodeParamValue(val)
		case "filename*":
			decoded, err := decodeRFC5987(val)
			if err == nil {
				cd.FilenameExt = decoded
			}
		}
	}

	return cd, nil
}

// decodeParamValue unquotes a quoted-string value or returns the bare value.
func decodeParamValue(v string) string {
	if len(v) >= 2 && v[0] == '"' {
		return decodeQuotedString(v)
	}
	return v
}

// decodeRFC5987 decodes a RFC 5987 encoded parameter value.
// Format: charset'language'encoded  (e.g., UTF-8''foo%20bar.txt)
func decodeRFC5987(v string) (string, error) {
	// Find first apostrophe (charset delimiter)
	first := strings.IndexByte(v, '\'')
	if first < 0 {
		return "", fmt.Errorf("headers: invalid RFC 5987 value: missing charset delimiter")
	}
	second := strings.IndexByte(v[first+1:], '\'')
	if second < 0 {
		return "", fmt.Errorf("headers: invalid RFC 5987 value: missing language delimiter")
	}

	encoded := v[first+1+second+1:]
	decoded, err := url.PathUnescape(encoded)
	if err != nil {
		return "", fmt.Errorf("headers: RFC 5987 percent-decode error: %w", err)
	}
	return decoded, nil
}

// PreferredFilename returns the best available filename.
// FilenameExt (RFC 5987) is preferred over Filename when both are present.
func (c ContentDisposition) PreferredFilename() string {
	if c.FilenameExt != "" {
		return c.FilenameExt
	}
	return c.Filename
}

// String returns the canonical header value.
func (c ContentDisposition) String() string {
	if c.Type == "" {
		return ""
	}

	var b strings.Builder
	b.WriteString(c.Type)

	if c.Name != "" {
		b.WriteString("; name=")
		b.WriteString(Quote(c.Name))
	}
	if c.Filename != "" {
		b.WriteString("; filename=")
		b.WriteString(Quote(c.Filename))
	}
	if c.FilenameExt != "" {
		b.WriteString("; filename*=UTF-8''")
		b.WriteString(url.PathEscape(c.FilenameExt))
	}

	return b.String()
}
