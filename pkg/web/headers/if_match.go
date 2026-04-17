package headers

import (
	"fmt"
	"strings"
)

// IfMatch represents a parsed If-Match header value.
// Strong ETags only (RFC 7232 3.1).
type IfMatch struct {
	Tags     []string // ETag values without quotes or W/ prefix (strong only)
	Wildcard bool     // true if value is "*"
}

// ParseIfMatch parses an If-Match header value.
// Weak ETags (W/"...") are rejected per RFC 7232 3.1.
func ParseIfMatch(v string) (IfMatch, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return IfMatch{}, fmt.Errorf("headers: empty If-Match")
	}

	if v == "*" {
		return IfMatch{Wildcard: true}, nil
	}

	parts := strings.Split(v, ",")
	tags := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.HasPrefix(strings.ToUpper(part), "W/") {
			return IfMatch{}, fmt.Errorf("headers: If-Match does not allow weak ETags: %q", part)
		}

		if len(part) < 2 || part[0] != '"' || part[len(part)-1] != '"' {
			return IfMatch{}, fmt.Errorf("headers: If-Match ETag must be quoted: %q", part)
		}

		tags = append(tags, part[1:len(part)-1])
	}

	if len(tags) == 0 {
		return IfMatch{}, fmt.Errorf("headers: If-Match contains no valid ETags")
	}

	return IfMatch{Tags: tags}, nil
}

// Matches reports whether the given ETag (unquoted, without W/) satisfies
// the If-Match condition using strong comparison.
// Wildcard matches any non-empty ETag.
func (m IfMatch) Matches(etag string) bool {
	if etag == "" {
		return false
	}
	if m.Wildcard {
		return true
	}
	for _, t := range m.Tags {
		if t == etag {
			return true
		}
	}
	return false
}

// String returns the canonical serialized form.
func (m IfMatch) String() string {
	if m.Wildcard {
		return "*"
	}
	parts := make([]string, len(m.Tags))
	for i, t := range m.Tags {
		parts[i] = `"` + t + `"`
	}
	return strings.Join(parts, ", ")
}
