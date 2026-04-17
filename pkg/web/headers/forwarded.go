package headers

import (
	"fmt"
	"strings"
)

// Forwarded represents a single forwarded-element from an RFC 7239 Forwarded header.
// A Forwarded header may contain multiple comma-separated elements.
type Forwarded struct {
	For   string // "for" parameter (host/IP, may be quoted)
	By    string // "by" parameter
	Host  string // "host" parameter
	Proto string // "proto" parameter
}

// ParseForwarded parses the value of an RFC 7239 Forwarded header.
// Multiple elements may be comma-separated; each element has key=value pairs separated by semicolons.
// Quoted-string values are unquoted. Unknown keys are silently ignored.
// Returns an error if the header value is structurally unparseable.
func ParseForwarded(v string) ([]Forwarded, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return []Forwarded{}, nil
	}

	elements := strings.Split(v, ",")
	result := make([]Forwarded, 0, len(elements))

	for _, elem := range elements {
		elem = strings.TrimSpace(elem)
		if elem == "" {
			continue
		}

		var fwd Forwarded
		pairs := strings.Split(elem, ";")
		for _, pair := range pairs {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}

			idx := strings.IndexByte(pair, '=')
			if idx < 0 {
				return nil, fmt.Errorf("headers: malformed forwarded element %q: missing '='", pair)
			}

			key := strings.ToLower(strings.TrimSpace(pair[:idx]))
			val := strings.TrimSpace(pair[idx+1:])

			if key == "" {
				return nil, fmt.Errorf("headers: malformed forwarded element: empty key in %q", pair)
			}

			if len(val) >= 2 && val[0] == '"' {
				val = decodeQuotedString(val)
			}

			switch key {
			case "for":
				fwd.For = val
			case "by":
				fwd.By = val
			case "host":
				fwd.Host = val
			case "proto":
				fwd.Proto = val
			// unknown keys are silently ignored
			}
		}

		result = append(result, fwd)
	}

	return result, nil
}
