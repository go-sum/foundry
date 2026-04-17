package headers

import (
	"sort"
	"strconv"
	"strings"
)

// Language is a single entry in an Accept-Language header.
type Language struct {
	Tag string  // BCP-47 language tag e.g. "en-US"
	Q   float64 // default 1.0
}

// AcceptLanguage represents a parsed Accept-Language header.
type AcceptLanguage struct {
	Languages []Language
}

// ParseAcceptLanguage parses an Accept-Language header value.
func ParseAcceptLanguage(v string) (AcceptLanguage, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return AcceptLanguage{}, nil
	}

	parts := strings.Split(v, ",")
	langs := make([]Language, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		segments := strings.Split(part, ";")
		tag := strings.TrimSpace(segments[0])
		if tag == "" {
			continue
		}

		q := 1.0
		for _, seg := range segments[1:] {
			seg = strings.TrimSpace(seg)
			lower := strings.ToLower(seg)
			if strings.HasPrefix(lower, "q=") {
				if f, err := strconv.ParseFloat(seg[2:], 64); err == nil {
					q = f
				}
			}
		}

		langs = append(langs, Language{Tag: tag, Q: q})
	}

	sort.SliceStable(langs, func(i, j int) bool {
		return langs[i].Q > langs[j].Q
	})

	return AcceptLanguage{Languages: langs}, nil
}

// Negotiate returns the best matching language from offered.
// Matching is prefix-based: "en" matches "en-US" and "en-GB".
func (a AcceptLanguage) Negotiate(offered ...string) string {
	if len(a.Languages) == 0 {
		if len(offered) > 0 {
			return offered[0]
		}
		return ""
	}

	for _, lang := range a.Languages {
		if lang.Q <= 0 {
			continue
		}
		prefix := strings.ToLower(lang.Tag)
		for _, o := range offered {
			lower := strings.ToLower(o)
			if lower == prefix {
				return o
			}
			// Prefix match: "en" matches "en-US"
			if strings.HasPrefix(lower, prefix+"-") {
				return o
			}
			// Also allow offered prefix matching Accept tag: "en-US" Accept matches offered "en"
			if strings.HasPrefix(prefix, lower+"-") {
				return o
			}
		}
	}

	return ""
}

// String returns the canonical serialized form.
func (a AcceptLanguage) String() string {
	parts := make([]string, len(a.Languages))
	for i, lang := range a.Languages {
		if lang.Q == 1.0 {
			parts[i] = lang.Tag
		} else {
			parts[i] = lang.Tag + ";q=" + strconv.FormatFloat(lang.Q, 'f', -1, 64)
		}
	}
	return strings.Join(parts, ", ")
}
