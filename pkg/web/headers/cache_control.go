package headers

import (
	"strconv"
	"strings"
)

var cacheControlIntSetters = map[string]func(*CacheControl, int){
	"max-age": func(cc *CacheControl, n int) { cc.MaxAge = &n },
	"s-maxage": func(cc *CacheControl, n int) {
		cc.SMaxAge = &n
	},
	"stale-while-revalidate": func(cc *CacheControl, n int) {
		cc.StaleWhileRevalidate = &n
	},
	"stale-if-error": func(cc *CacheControl, n int) { cc.StaleIfError = &n },
	"max-stale": func(cc *CacheControl, n int) {
		cc.MaxStale = &n
	},
	"min-fresh": func(cc *CacheControl, n int) { cc.MinFresh = &n },
}

var cacheControlFlagSetters = map[string]func(*CacheControl){
	"no-cache": func(cc *CacheControl) { cc.NoCache = true },
	"no-store": func(cc *CacheControl) { cc.NoStore = true },
	"no-transform": func(cc *CacheControl) {
		cc.NoTransform = true
	},
	"must-revalidate": func(cc *CacheControl) { cc.MustRevalidate = true },
	"proxy-revalidate": func(cc *CacheControl) {
		cc.ProxyRevalidate = true
	},
	"must-understand": func(cc *CacheControl) { cc.MustUnderstand = true },
	"private":         func(cc *CacheControl) { cc.Private = true },
	"public":          func(cc *CacheControl) { cc.Public = true },
	"immutable": func(cc *CacheControl) {
		cc.Immutable = true
	},
	"only-if-cached": func(cc *CacheControl) { cc.OnlyIfCached = true },
	"max-stale": func(cc *CacheControl) {
		// max-stale without value means any stale is accepted.
		zero := 0
		cc.MaxStale = &zero
	},
}

// CacheControl represents a parsed Cache-Control header (request or response).
// All duration fields are in seconds. nil pointer = directive not present.
type CacheControl struct {
	// Response directives
	MaxAge               *int
	SMaxAge              *int
	NoCache              bool
	NoStore              bool
	NoTransform          bool
	MustRevalidate       bool
	ProxyRevalidate      bool
	MustUnderstand       bool
	Private              bool
	Public               bool
	Immutable            bool
	StaleWhileRevalidate *int
	StaleIfError         *int

	// Request directives
	MaxStale     *int // nil = no max-stale; 0 = any stale accepted; N = up to N seconds stale
	MinFresh     *int
	OnlyIfCached bool
}

// ParseCacheControl parses a Cache-Control header value.
func ParseCacheControl(v string) (CacheControl, error) {
	var cc CacheControl
	v = strings.TrimSpace(v)
	if v == "" {
		return cc, nil
	}

	parts := strings.Split(v, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		lower := strings.ToLower(part)

		// Check for token=value form
		idx := strings.IndexByte(lower, '=')
		if idx >= 0 {
			directive := strings.TrimSpace(lower[:idx])
			valStr := strings.TrimSpace(lower[idx+1:])
			n, err := strconv.Atoi(valStr)
			if err != nil {
				// Skip malformed integer directives.
				continue
			}
			if apply, ok := cacheControlIntSetters[directive]; ok {
				apply(&cc, n)
			}
			continue
		}

		if apply, ok := cacheControlFlagSetters[lower]; ok {
			apply(&cc)
		}
	}

	return cc, nil
}

// String serializes the CacheControl to its canonical header string.
func (c CacheControl) String() string {
	var parts []string

	if c.Public {
		parts = append(parts, "public")
	}
	if c.Private {
		parts = append(parts, "private")
	}
	if c.MaxAge != nil {
		parts = append(parts, "max-age="+strconv.Itoa(*c.MaxAge))
	}
	if c.SMaxAge != nil {
		parts = append(parts, "s-maxage="+strconv.Itoa(*c.SMaxAge))
	}
	if c.NoCache {
		parts = append(parts, "no-cache")
	}
	if c.NoStore {
		parts = append(parts, "no-store")
	}
	if c.NoTransform {
		parts = append(parts, "no-transform")
	}
	if c.MustRevalidate {
		parts = append(parts, "must-revalidate")
	}
	if c.ProxyRevalidate {
		parts = append(parts, "proxy-revalidate")
	}
	if c.MustUnderstand {
		parts = append(parts, "must-understand")
	}
	if c.Immutable {
		parts = append(parts, "immutable")
	}
	if c.StaleWhileRevalidate != nil {
		parts = append(parts, "stale-while-revalidate="+strconv.Itoa(*c.StaleWhileRevalidate))
	}
	if c.StaleIfError != nil {
		parts = append(parts, "stale-if-error="+strconv.Itoa(*c.StaleIfError))
	}
	if c.OnlyIfCached {
		parts = append(parts, "only-if-cached")
	}
	if c.MaxStale != nil {
		parts = append(parts, "max-stale="+strconv.Itoa(*c.MaxStale))
	}
	if c.MinFresh != nil {
		parts = append(parts, "min-fresh="+strconv.Itoa(*c.MinFresh))
	}

	return strings.Join(parts, ", ")
}
