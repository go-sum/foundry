package headers

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"
)

// ByteRange is a single requested byte range.
// If Start is nil, it is a suffix range (last N bytes).
// If End is nil, it extends to the end of the resource.
type ByteRange struct {
	Start *int64 // nil = suffix range (from end)
	End   *int64 // nil = to EOF
}

// Range represents a parsed Range header.
type Range struct {
	Unit   string      // "bytes"
	Ranges []ByteRange
}

// ParseRange parses a Range header value.
// Returns an error for empty unit, empty ranges, or malformed ranges.
func ParseRange(v string) (Range, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return Range{}, fmt.Errorf("headers: empty Range")
	}

	eqIdx := strings.IndexByte(v, '=')
	if eqIdx < 0 {
		return Range{}, fmt.Errorf("headers: Range missing '=': %q", v)
	}

	unit := strings.ToLower(strings.TrimSpace(v[:eqIdx]))
	if unit == "" {
		return Range{}, fmt.Errorf("headers: Range missing unit")
	}

	rangeSpec := strings.TrimSpace(v[eqIdx+1:])
	if rangeSpec == "" {
		return Range{}, fmt.Errorf("headers: Range missing ranges")
	}

	parts := strings.Split(rangeSpec, ",")
	ranges := make([]ByteRange, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		dashIdx := strings.IndexByte(part, '-')
		if dashIdx < 0 {
			return Range{}, fmt.Errorf("headers: Range invalid range spec: %q", part)
		}

		startStr := strings.TrimSpace(part[:dashIdx])
		endStr := strings.TrimSpace(part[dashIdx+1:])

		var br ByteRange

		if startStr == "" {
			// Suffix range: -N (last N bytes). End holds the suffix length.
			if endStr == "" {
				return Range{}, fmt.Errorf("headers: Range invalid range spec: %q", part)
			}
			n, err := strconv.ParseInt(endStr, 10, 64)
			if err != nil {
				return Range{}, fmt.Errorf("headers: Range invalid suffix length: %w", err)
			}
			br.End = &n // Start remains nil to signal suffix range
		} else {
			s, err := strconv.ParseInt(startStr, 10, 64)
			if err != nil {
				return Range{}, fmt.Errorf("headers: Range invalid start: %w", err)
			}
			br.Start = &s
			if endStr != "" {
				e, err := strconv.ParseInt(endStr, 10, 64)
				if err != nil {
					return Range{}, fmt.Errorf("headers: Range invalid end: %w", err)
				}
				br.End = &e
			}
		}

		ranges = append(ranges, br)
	}

	if len(ranges) == 0 {
		return Range{}, fmt.Errorf("headers: Range contains no valid ranges")
	}

	return Range{Unit: unit, Ranges: ranges}, nil
}

// CanSatisfy reports whether at least one range can be satisfied for a
// resource of the given size.
func (r Range) CanSatisfy(size int64) bool {
	for _, br := range r.Ranges {
		if br.Start == nil {
			// Suffix range.
			if br.End != nil && *br.End > 0 && size > 0 {
				return true
			}
		} else {
			start := *br.Start
			if start >= size {
				continue
			}
			return true
		}
	}
	return false
}

// Normalize resolves the first range against a resource of the given size,
// returning the clamped [start, end] (inclusive) or ok=false if not satisfiable.
func (r Range) Normalize(size int64) (start, end int64, ok bool) {
	if len(r.Ranges) == 0 {
		return 0, 0, false
	}

	br := r.Ranges[0]

	if br.Start == nil {
		// Suffix range.
		if br.End == nil || *br.End == 0 {
			return 0, 0, false
		}
		suffixLen := *br.End
		start = size - suffixLen
		if start < 0 {
			start = 0
		}
		end = size - 1
		if end < 0 {
			return 0, 0, false
		}
		return start, end, true
	}

	start = *br.Start
	if start >= size {
		return 0, 0, false
	}

	if br.End == nil {
		end = size - 1
	} else {
		end = *br.End
		if end >= size {
			end = size - 1
		}
	}

	if start > end {
		return 0, 0, false
	}

	return start, end, true
}

// String returns the canonical "bytes=..." form.
func (r Range) String() string {
	unit := cmp.Or(r.Unit, "bytes")

	parts := make([]string, len(r.Ranges))
	for i, br := range r.Ranges {
		if br.Start == nil {
			// Suffix range.
			if br.End != nil {
				parts[i] = fmt.Sprintf("-%d", *br.End)
			}
		} else if br.End == nil {
			parts[i] = fmt.Sprintf("%d-", *br.Start)
		} else {
			parts[i] = fmt.Sprintf("%d-%d", *br.Start, *br.End)
		}
	}

	return unit + "=" + strings.Join(parts, ", ")
}
