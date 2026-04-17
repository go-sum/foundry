package headers

import "strings"

// Vary represents a parsed Vary header.
type Vary struct {
	Star   bool     // true if value is "*"
	Fields []string // canonical, case-insensitive
}

// ParseVary parses a Vary header value.
func ParseVary(v string) Vary {
	v = strings.TrimSpace(v)
	if v == "" {
		return Vary{}
	}

	if v == "*" {
		return Vary{Star: true}
	}

	parts := strings.Split(v, ",")
	seen := make(map[string]bool, len(parts))
	fields := make([]string, 0, len(parts))

	for _, part := range parts {
		f := strings.TrimSpace(part)
		if f == "" {
			continue
		}
		canonical := CanonicalName(f)
		lower := strings.ToLower(canonical)
		if !seen[lower] {
			seen[lower] = true
			fields = append(fields, canonical)
		}
	}

	return Vary{Fields: fields}
}

// Add adds a field to the Vary set (deduplicated, case-insensitive).
// Returns a new Vary value (immutable design).
func (v Vary) Add(field string) Vary {
	if v.Star {
		return v
	}

	canonical := CanonicalName(field)
	lower := strings.ToLower(canonical)

	// Check if already present.
	for _, f := range v.Fields {
		if strings.ToLower(f) == lower {
			return v
		}
	}

	newFields := make([]string, len(v.Fields)+1)
	copy(newFields, v.Fields)
	newFields[len(v.Fields)] = canonical

	return Vary{Fields: newFields}
}

// Has reports whether field is in the Vary set (case-insensitive).
func (v Vary) Has(field string) bool {
	if v.Star {
		return true
	}
	lower := strings.ToLower(field)
	for _, f := range v.Fields {
		if strings.ToLower(f) == lower {
			return true
		}
	}
	return false
}

// String returns the canonical serialized form.
// Fields are emitted in the order they were added. "*" is emitted alone.
func (v Vary) String() string {
	if v.Star {
		return "*"
	}
	return strings.Join(v.Fields, ", ")
}
