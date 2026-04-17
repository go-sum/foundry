package headers

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// IfRange represents a parsed If-Range header.
// Contains either an ETag (strong only) or an HTTP-date.
type IfRange struct {
	ETag string    // strong ETag (unquoted, without W/)
	Date time.Time // zero if ETag was used
}

// ParseIfRange parses an If-Range header value.
// If the value looks like an HTTP-date (contains spaces or commas), it is
// parsed as a date; otherwise it is treated as an ETag.
// Weak ETags (W/"...") are invalid in If-Range (RFC 7233 3.2).
func ParseIfRange(v string) (IfRange, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return IfRange{}, fmt.Errorf("headers: empty If-Range")
	}

	// Weak ETags are explicitly forbidden in If-Range.
	if strings.HasPrefix(strings.ToUpper(v), "W/") {
		return IfRange{}, fmt.Errorf("headers: If-Range does not allow weak ETags: %q", v)
	}

	// If it looks like a quoted ETag, parse as ETag.
	if len(v) >= 2 && v[0] == '"' {
		if v[len(v)-1] != '"' {
			return IfRange{}, fmt.Errorf("headers: If-Range ETag not properly quoted: %q", v)
		}
		return IfRange{ETag: v[1 : len(v)-1]}, nil
	}

	// Otherwise, attempt to parse as HTTP-date.
	// Try known HTTP date formats.
	formats := []string{
		http.TimeFormat,                  // "Mon, 02 Jan 2006 15:04:05 GMT"
		"Monday, 02-Jan-06 15:04:05 MST", // RFC 850
		"Mon Jan  2 15:04:05 2006",       // ANSI C
	}
	for _, format := range formats {
		if t, err := time.Parse(format, v); err == nil {
			return IfRange{Date: t.UTC()}, nil
		}
	}

	return IfRange{}, fmt.Errorf("headers: If-Range value is not a valid ETag or HTTP-date: %q", v)
}

// Matches reports whether the given representation satisfies the If-Range condition.
// etag should be the current strong ETag (unquoted). modTime is the Last-Modified time.
// A weak ETag never satisfies If-Range (RFC 7233 3.2).
func (r IfRange) Matches(etag string, modTime time.Time) bool {
	if r.ETag != "" {
		return r.ETag == etag
	}
	if !r.Date.IsZero() {
		// Date comparison: resource must not have been modified after the If-Range date.
		return !modTime.After(r.Date)
	}
	return false
}

// String returns the canonical serialized form.
func (r IfRange) String() string {
	if r.ETag != "" {
		return `"` + r.ETag + `"`
	}
	if !r.Date.IsZero() {
		return r.Date.UTC().Format(http.TimeFormat)
	}
	return ""
}
