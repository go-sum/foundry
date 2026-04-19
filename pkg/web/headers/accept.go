package headers

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// MediaType represents a single entry in an Accept header.
type MediaType struct {
	Type   string            // e.g. "text"
	Sub    string            // e.g. "html" or "*"
	Q      float64           // quality factor [0,1], default 1.0
	Params map[string]string // extension parameters
}

// String returns the canonical representation of this media type.
func (m MediaType) String() string {
	mt := m.Type + "/" + m.Sub
	var b strings.Builder
	b.WriteString(mt)
	if m.Q != 1.0 {
		b.WriteString(fmt.Sprintf(";q=%g", m.Q))
	}
	for k, v := range m.Params {
		b.WriteString(";" + k + "=" + Quote(v))
	}
	return b.String()
}

// Accept represents a parsed Accept header value.
type Accept struct {
	MediaTypes []MediaType
}

// ParseAccept parses an Accept header value.
// Invalid entries are silently skipped.
func ParseAccept(v string) (Accept, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return Accept{}, nil
	}

	parts := strings.Split(v, ",")
	mts := make([]MediaType, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split into media-range and params
		segments := strings.Split(part, ";")
		mediaRange := strings.TrimSpace(segments[0])

		slash := strings.IndexByte(mediaRange, '/')
		if slash < 0 {
			continue
		}

		typ := strings.ToLower(strings.TrimSpace(mediaRange[:slash]))
		sub := strings.ToLower(strings.TrimSpace(mediaRange[slash+1:]))
		if typ == "" || sub == "" {
			continue
		}

		q := 1.0
		params := make(map[string]string)

		for _, seg := range segments[1:] {
			seg = strings.TrimSpace(seg)
			if seg == "" {
				continue
			}
			idx := strings.IndexByte(seg, '=')
			if idx < 0 {
				continue
			}
			pk := strings.ToLower(strings.TrimSpace(seg[:idx]))
			pv := strings.TrimSpace(seg[idx+1:])
			if pk == "q" {
				if f, err := strconv.ParseFloat(pv, 64); err == nil {
					q = f
				}
			} else {
				params[pk] = pv
			}
		}

		mt := MediaType{Type: typ, Sub: sub, Q: q}
		if len(params) > 0 {
			mt.Params = params
		}
		mts = append(mts, mt)
	}

	// Stable sort by q descending — equal-q entries keep input order.
	sort.SliceStable(mts, func(i, j int) bool {
		return mts[i].Q > mts[j].Q
	})

	return Accept{MediaTypes: mts}, nil
}

// Negotiate returns the best matching media type from offered for this Accept value.
// It returns "" if none of the offered types are acceptable (q=0 excluded).
// offered entries should be full media types like "text/html" or "application/json".
func (a Accept) Negotiate(offered ...string) string {
	// Empty Accept means accept anything.
	if len(a.MediaTypes) == 0 {
		if len(offered) > 0 {
			return offered[0]
		}
		return ""
	}

	type candidate struct {
		offer      string
		q          float64
		specificity int // 2=exact, 1=type/*, 0=*/*
	}

	var best candidate
	found := false

	for _, offer := range offered {
		lower := strings.ToLower(offer)
		slash := strings.IndexByte(lower, '/')
		if slash < 0 {
			continue
		}
		otype := lower[:slash]
		osub := lower[slash+1:]

		// Find best matching Accept entry for this offer.
		var matchQ float64
		var matchSpec int
		matched := false

		for _, mt := range a.MediaTypes {
			var spec int
			switch {
			case mt.Type == otype && mt.Sub == osub:
				spec = 2
			case mt.Type == otype && mt.Sub == "*":
				spec = 1
			case mt.Type == "*" && mt.Sub == "*":
				spec = 0
			default:
				continue
			}
			if !matched || spec > matchSpec {
				matchQ = mt.Q
				matchSpec = spec
				matched = true
			}
		}

		if !matched || matchQ <= 0 {
			continue
		}

		if !found || matchQ > best.q || (matchQ == best.q && matchSpec > best.specificity) {
			best = candidate{offer: offer, q: matchQ, specificity: matchSpec}
			found = true
		}
	}

	if !found {
		return ""
	}
	return best.offer
}

// String returns the canonical serialized form.
func (a Accept) String() string {
	parts := make([]string, len(a.MediaTypes))
	for i, mt := range a.MediaTypes {
		parts[i] = mt.String()
	}
	return strings.Join(parts, ", ")
}
