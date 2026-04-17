package headers

import (
	"fmt"
	"strings"
)

// IfNoneMatch represents a parsed If-None-Match header.
// Supports both strong and weak ETags.
type IfNoneMatch struct {
	Tags     []string // strong ETag values (unquoted)
	Weak     []string // weak ETag values (unquoted, without W/)
	Wildcard bool
}

// ParseIfNoneMatch parses an If-None-Match header value.
func ParseIfNoneMatch(v string) (IfNoneMatch, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return IfNoneMatch{}, fmt.Errorf("headers: empty If-None-Match")
	}

	if v == "*" {
		return IfNoneMatch{Wildcard: true}, nil
	}

	parts := strings.Split(v, ",")
	var strong, weak []string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		isWeak := false
		if len(part) >= 2 && strings.ToUpper(part[:2]) == "W/" {
			isWeak = true
			part = part[2:]
		}

		if len(part) < 2 || part[0] != '"' || part[len(part)-1] != '"' {
			return IfNoneMatch{}, fmt.Errorf("headers: If-None-Match ETag must be quoted: %q", part)
		}

		tag := part[1 : len(part)-1]
		if isWeak {
			weak = append(weak, tag)
		} else {
			strong = append(strong, tag)
		}
	}

	if len(strong) == 0 && len(weak) == 0 {
		return IfNoneMatch{}, fmt.Errorf("headers: If-None-Match contains no valid ETags")
	}

	return IfNoneMatch{Tags: strong, Weak: weak}, nil
}

// Matches reports whether the given ETag satisfies the If-None-Match condition.
// When weak=true, uses weak comparison (W/"x" matches "x").
// Wildcard matches any ETag.
func (m IfNoneMatch) Matches(etag string, weak bool) bool {
	if m.Wildcard {
		return true
	}
	// Strong comparison: strong tags must match exactly.
	for _, t := range m.Tags {
		if t == etag {
			return true
		}
	}
	if weak {
		// Weak comparison: weak tags match by value.
		for _, t := range m.Weak {
			if t == etag {
				return true
			}
		}
	}
	return false
}

// String returns the canonical serialized form.
func (m IfNoneMatch) String() string {
	if m.Wildcard {
		return "*"
	}
	var parts []string
	for _, t := range m.Tags {
		parts = append(parts, `"`+t+`"`)
	}
	for _, t := range m.Weak {
		parts = append(parts, `W/"`+t+`"`)
	}
	return strings.Join(parts, ", ")
}
