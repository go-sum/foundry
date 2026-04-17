package headers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var bareSetCookieAttrs = map[string]func(*SetCookie){
	"httponly": func(sc *SetCookie) { sc.HttpOnly = true },
	"secure": func(sc *SetCookie) {
		sc.Secure = true
	},
	"partitioned": func(sc *SetCookie) { sc.Partitioned = true },
}

var setCookieSameSiteValues = map[string]string{
	"strict": "Strict",
	"lax":    "Lax",
	"none":   "None",
}

var setCookiePriorityValues = map[string]string{
	"low":    "Low",
	"medium": "Medium",
	"high":   "High",
}

// SetCookie represents a parsed or constructed Set-Cookie header value.
type SetCookie struct {
	Name        string
	Value       string
	Domain      string
	Expires     time.Time // zero = not set
	HttpOnly    bool
	MaxAge      *int // nil = not set; ptr to 0 = delete; ptr to N = TTL in seconds
	Partitioned bool
	Path        string
	Priority    string // "Low" | "Medium" | "High" | ""
	SameSite    string // "Strict" | "Lax" | "None" | ""
	Secure      bool
}

// ParseSetCookie parses a Set-Cookie header value.
func ParseSetCookie(v string) (SetCookie, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return SetCookie{}, fmt.Errorf("headers: empty Set-Cookie")
	}

	var sc SetCookie

	parts := strings.Split(v, ";")

	// First part is name=value.
	first := strings.TrimSpace(parts[0])
	eqIdx := strings.IndexByte(first, '=')
	if eqIdx < 0 {
		return SetCookie{}, fmt.Errorf("headers: Set-Cookie missing name=value: %q", first)
	}
	sc.Name = strings.TrimSpace(first[:eqIdx])
	sc.Value = strings.TrimSpace(first[eqIdx+1:])

	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		lower := strings.ToLower(part)

		if apply, ok := bareSetCookieAttrs[lower]; ok {
			apply(&sc)
			continue
		}

		idx := strings.IndexByte(part, '=')
		if idx < 0 {
			continue
		}

		attr := strings.ToLower(strings.TrimSpace(part[:idx]))
		val := strings.TrimSpace(part[idx+1:])

		switch attr {
		case "domain":
			sc.Domain = strings.ToLower(val)
		case "path":
			sc.Path = val
		case "samesite":
			if sameSite, ok := setCookieSameSiteValues[strings.ToLower(val)]; ok {
				sc.SameSite = sameSite
			}
		case "priority":
			if priority, ok := setCookiePriorityValues[strings.ToLower(val)]; ok {
				sc.Priority = priority
			}
		case "max-age":
			n, err := strconv.Atoi(val)
			if err == nil {
				sc.MaxAge = &n
			}
		case "expires":
			if t, err := time.Parse(http.TimeFormat, val); err == nil {
				sc.Expires = t.UTC()
			} else if t, err := time.Parse("Monday, 02-Jan-06 15:04:05 MST", val); err == nil {
				sc.Expires = t.UTC()
			}
		}
	}

	return sc, nil
}

// String serializes the SetCookie to its canonical header value.
// Attributes are emitted in a fixed order:
// Domain, Expires, HttpOnly, Max-Age, Partitioned, Path, Priority, SameSite, Secure.
// Partitioned forces Secure=true.
// SameSite defaults to "Lax" if empty.
func (s SetCookie) String() string {
	var b strings.Builder
	b.WriteString(s.Name)
	b.WriteByte('=')
	b.WriteString(s.Value)

	if s.Domain != "" {
		b.WriteString("; Domain=")
		b.WriteString(s.Domain)
	}
	if !s.Expires.IsZero() {
		b.WriteString("; Expires=")
		b.WriteString(s.Expires.UTC().Format(http.TimeFormat))
	}
	if s.HttpOnly {
		b.WriteString("; HttpOnly")
	}
	if s.MaxAge != nil {
		b.WriteString("; Max-Age=")
		b.WriteString(strconv.Itoa(*s.MaxAge))
	}
	if s.Partitioned {
		b.WriteString("; Partitioned")
	}
	if s.Path != "" {
		b.WriteString("; Path=")
		b.WriteString(s.Path)
	}
	if s.Priority != "" {
		b.WriteString("; Priority=")
		b.WriteString(s.Priority)
	}

	// SameSite defaults to Lax.
	sameSite := s.SameSite
	if sameSite == "" {
		sameSite = "Lax"
	}
	b.WriteString("; SameSite=")
	b.WriteString(sameSite)

	// Partitioned forces Secure.
	if s.Secure || s.Partitioned {
		b.WriteString("; Secure")
	}

	return b.String()
}
