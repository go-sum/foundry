package headers

import (
	"sort"
	"strconv"
	"strings"
)

// Encoding is a single entry in an Accept-Encoding header.
type Encoding struct {
	Token string  // e.g. "gzip", "br", "identity", "*"
	Q     float64 // default 1.0; 0 = explicitly rejected
}

// AcceptEncoding represents a parsed Accept-Encoding header.
type AcceptEncoding struct {
	Encodings []Encoding
}

// ParseAcceptEncoding parses an Accept-Encoding header value.
func ParseAcceptEncoding(v string) (AcceptEncoding, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return AcceptEncoding{}, nil
	}

	parts := strings.Split(v, ",")
	encs := make([]Encoding, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		segments := strings.Split(part, ";")
		token := strings.ToLower(strings.TrimSpace(segments[0]))
		if token == "" {
			continue
		}

		q := 1.0
		for _, seg := range segments[1:] {
			seg = strings.TrimSpace(seg)
			if strings.HasPrefix(strings.ToLower(seg), "q=") {
				if f, err := strconv.ParseFloat(seg[2:], 64); err == nil {
					q = f
				}
			}
		}

		encs = append(encs, Encoding{Token: token, Q: q})
	}

	sort.SliceStable(encs, func(i, j int) bool {
		return encs[i].Q > encs[j].Q
	})

	return AcceptEncoding{Encodings: encs}, nil
}

// Negotiate returns the best acceptable encoding from the offered list
// in preference order (e.g., ["br", "gzip", "deflate", "identity"]).
// Returns "" if all offered encodings are explicitly rejected (q=0)
// INCLUDING identity, which means the client rejected all representations.
// If identity is not listed, it is implicitly acceptable at q=1 unless
// the Accept-Encoding header contains "identity;q=0" or "*;q=0".
func (a AcceptEncoding) Negotiate(offered ...string) string {
	if len(a.Encodings) == 0 {
		// Empty header: identity is acceptable.
		for _, o := range offered {
			if strings.ToLower(o) == "identity" {
				return o
			}
		}
		// Return first offered if identity not in list — all are acceptable.
		if len(offered) > 0 {
			return offered[0]
		}
		return ""
	}

	// Build lookup map.
	qmap := make(map[string]float64, len(a.Encodings))
	hasWildcard := false
	wildcardQ := 0.0

	for _, enc := range a.Encodings {
		if enc.Token == "*" {
			hasWildcard = true
			wildcardQ = enc.Q
		} else {
			qmap[enc.Token] = enc.Q
		}
	}

	getQ := func(token string) (float64, bool) {
		if q, ok := qmap[token]; ok {
			return q, true
		}
		if hasWildcard {
			return wildcardQ, true
		}
		// identity is implicitly acceptable if not explicitly listed and no wildcard.
		if token == "identity" {
			return 1.0, true
		}
		return 0, false
	}

	for _, o := range offered {
		lower := strings.ToLower(o)
		q, ok := getQ(lower)
		if !ok || q <= 0 {
			continue
		}
		return o
	}

	return ""
}

// String returns the canonical serialized form.
func (a AcceptEncoding) String() string {
	parts := make([]string, len(a.Encodings))
	for i, enc := range a.Encodings {
		if enc.Q == 1.0 {
			parts[i] = enc.Token
		} else {
			parts[i] = enc.Token + ";q=" + strconv.FormatFloat(enc.Q, 'f', -1, 64)
		}
	}
	return strings.Join(parts, ", ")
}
