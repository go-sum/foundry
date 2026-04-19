package headers

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"
)

// ContentRange represents a parsed Content-Range header.
// Start and End are -1 for unsatisfied ranges ("bytes */size").
// Size is -1 for unknown ("bytes 0-100/*").
type ContentRange struct {
	Unit  string // "bytes"
	Start int64  // -1 if not applicable
	End   int64  // -1 if not applicable
	Size  int64  // -1 if unknown ("*")
}

// ParseContentRange parses a Content-Range header value.
// Accepts "bytes 0-499/1234", "bytes */1234", "bytes 0-499/*".
func ParseContentRange(v string) (ContentRange, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return ContentRange{}, fmt.Errorf("headers: empty Content-Range")
	}

	// Split unit from range spec.
	spaceIdx := strings.IndexByte(v, ' ')
	if spaceIdx < 0 {
		return ContentRange{}, fmt.Errorf("headers: Content-Range missing unit: %q", v)
	}

	unit := strings.ToLower(v[:spaceIdx])
	spec := strings.TrimSpace(v[spaceIdx+1:])

	// Split range from size on "/".
	slashIdx := strings.IndexByte(spec, '/')
	if slashIdx < 0 {
		return ContentRange{}, fmt.Errorf("headers: Content-Range missing '/': %q", spec)
	}

	rangeStr := strings.TrimSpace(spec[:slashIdx])
	sizeStr := strings.TrimSpace(spec[slashIdx+1:])

	var start, end, size int64

	// Parse size.
	if sizeStr == "*" {
		size = -1
	} else {
		n, err := strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			return ContentRange{}, fmt.Errorf("headers: Content-Range invalid size %q: %w", sizeStr, err)
		}
		size = n
	}

	// Parse range.
	if rangeStr == "*" {
		start = -1
		end = -1
	} else {
		dashIdx := strings.IndexByte(rangeStr, '-')
		if dashIdx < 0 {
			return ContentRange{}, fmt.Errorf("headers: Content-Range invalid range %q", rangeStr)
		}
		s, err := strconv.ParseInt(strings.TrimSpace(rangeStr[:dashIdx]), 10, 64)
		if err != nil {
			return ContentRange{}, fmt.Errorf("headers: Content-Range invalid start: %w", err)
		}
		e, err := strconv.ParseInt(strings.TrimSpace(rangeStr[dashIdx+1:]), 10, 64)
		if err != nil {
			return ContentRange{}, fmt.Errorf("headers: Content-Range invalid end: %w", err)
		}
		start = s
		end = e
	}

	return ContentRange{Unit: unit, Start: start, End: end, Size: size}, nil
}

// String returns the canonical "bytes start-end/size" form.
func (c ContentRange) String() string {
	unit := cmp.Or(c.Unit, "bytes")

	var rangeStr string
	if c.Start < 0 || c.End < 0 {
		rangeStr = "*"
	} else {
		rangeStr = fmt.Sprintf("%d-%d", c.Start, c.End)
	}

	var sizeStr string
	if c.Size < 0 {
		sizeStr = "*"
	} else {
		sizeStr = strconv.FormatInt(c.Size, 10)
	}

	return unit + " " + rangeStr + "/" + sizeStr
}
