package headers

import "strings"

// CookiePair is a single name=value cookie from a request Cookie header.
type CookiePair struct {
	Name  string
	Value string
}

// CookieList represents a parsed request Cookie header.
// Preserves order; allows duplicate names (returns first match on Get).
type CookieList struct {
	pairs []CookiePair
}

// ParseCookieList parses a request Cookie header value.
// Values are not percent-decoded (per RFC 6265).
func ParseCookieList(v string) CookieList {
	v = strings.TrimSpace(v)
	if v == "" {
		return CookieList{}
	}

	parts := strings.Split(v, ";")
	pairs := make([]CookiePair, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		idx := strings.IndexByte(part, '=')
		if idx < 0 {
			// Name-only cookie (no value).
			name := strings.TrimSpace(part)
			if name != "" {
				pairs = append(pairs, CookiePair{Name: name, Value: ""})
			}
			continue
		}

		name := strings.TrimSpace(part[:idx])
		value := strings.TrimSpace(part[idx+1:])
		if name != "" {
			pairs = append(pairs, CookiePair{Name: name, Value: value})
		}
	}

	return CookieList{pairs: pairs}
}

// Get returns the value of the first cookie with the given name.
func (c CookieList) Get(name string) (string, bool) {
	for _, p := range c.pairs {
		if p.Name == name {
			return p.Value, true
		}
	}
	return "", false
}

// Has reports whether a cookie with the given name exists.
func (c CookieList) Has(name string) bool {
	for _, p := range c.pairs {
		if p.Name == name {
			return true
		}
	}
	return false
}

// All returns all cookie pairs in order.
func (c CookieList) All() []CookiePair {
	if len(c.pairs) == 0 {
		return nil
	}
	out := make([]CookiePair, len(c.pairs))
	copy(out, c.pairs)
	return out
}

// String serializes the cookie list back to a Cookie: header value.
func (c CookieList) String() string {
	parts := make([]string, len(c.pairs))
	for i, p := range c.pairs {
		parts[i] = p.Name + "=" + p.Value
	}
	return strings.Join(parts, "; ")
}
