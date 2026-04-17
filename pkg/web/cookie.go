package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

var cookieSameSiteAttr = map[string]string{
	"":       "Lax",
	"Lax":    "Lax",
	"Strict": "Strict",
	"None":   "None",
}

// Cookie represents an HTTP cookie.
//
// MaxAge uses the Go stdlib convention:
//   - MaxAge == 0: omit Max-Age (session cookie)
//   - MaxAge > 0: set Max-Age to N seconds
//   - MaxAge < 0: set Max-Age=0 (instruct browser to delete the cookie)
type Cookie struct {
	Name        string
	Value       string
	Path        string
	Domain      string
	MaxAge      int
	Expires     time.Time
	Secure      bool
	HTTPOnly    bool
	SameSite    string // "Strict", "Lax", "None", or "" (default: Lax)
	Partitioned bool
	Priority    string // "Low", "Medium", "High", or ""
}

// isValidCookieName reports whether s is a valid RFC 6265 cookie name.
// It must be a non-empty US-ASCII string without CTL chars or separators.
func isValidCookieName(s string) bool {
	if s == "" {
		return false
	}
	for _, b := range []byte(s) {
		if b <= 0x20 || b >= 0x7F {
			return false
		}
		// RFC 6265 separator set
		switch b {
		case '(', ')', '<', '>', '@', ',', ';', ':', '\\', '"',
			'/', '[', ']', '?', '=', '{', '}', ' ', '\t':
			return false
		}
	}
	return true
}

// cookieSafeValue reports whether every byte in v is a valid bare cookie-value
// byte per RFC 6265 4.1.1 (US-ASCII visible chars excluding DQUOTE, semicolon,
// comma, and backslash).
func cookieSafeValue(v string) bool {
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c <= 0x20 || c >= 0x7f || c == '"' || c == ';' || c == ',' || c == '\\' {
			return false
		}
	}
	return true
}

// quoteValue returns v as a safe cookie value. If v is already safe per RFC 6265
// 4.1.1, it is returned as-is. Otherwise it is wrapped in DQUOTE with interior
// DQUOTE chars escaped as \".
func quoteValue(v string) string {
	if cookieSafeValue(v) {
		return v
	}
	escaped := strings.ReplaceAll(v, `"`, `\"`)
	return `"` + escaped + `"`
}

// Validate checks that the cookie fields satisfy RFC 6265 requirements and
// the additional enforcement rules for __Host-, __Secure-, and Partitioned cookies.
// Returns an error describing the first violation found.
func (c Cookie) Validate() error {
	if !isValidCookieName(c.Name) {
		return fmt.Errorf("web: cookie name %q is not a valid RFC 6265 token", c.Name)
	}
	if strings.ContainsAny(c.Name, "\r\n") {
		return fmt.Errorf("web: cookie name contains CR or LF")
	}
	if strings.ContainsAny(c.Value, "\r\n") {
		return fmt.Errorf("web: cookie value contains CR or LF")
	}
	if strings.ContainsAny(c.Path, "\r\n") {
		return fmt.Errorf("web: cookie path contains CR or LF")
	}
	if strings.ContainsAny(c.Domain, "\r\n") {
		return fmt.Errorf("web: cookie domain contains CR or LF")
	}
	if strings.HasPrefix(c.Name, "__Host-") {
		if !c.Secure {
			return fmt.Errorf("web: __Host- cookie %q requires Secure=true", c.Name)
		}
		if c.Path != "/" {
			return fmt.Errorf("web: __Host- cookie %q requires Path=/", c.Name)
		}
		if c.Domain != "" {
			return fmt.Errorf("web: __Host- cookie %q must not have a Domain attribute", c.Name)
		}
	}
	if strings.HasPrefix(c.Name, "__Secure-") {
		if !c.Secure {
			return fmt.Errorf("web: __Secure- cookie %q requires Secure=true", c.Name)
		}
	}
	return nil
}

// String formats the cookie as a Set-Cookie header value.
func (c Cookie) String() string {
	// Sanitize CRLF from all string fields defensively.
	name := sanitizeHeaderValue("cookie-name", c.Name)
	value := sanitizeHeaderValue("cookie-value", c.Value)
	path := sanitizeHeaderValue("cookie-path", c.Path)
	domain := sanitizeHeaderValue("cookie-domain", c.Domain)

	// Enforce __Host- prefix constraints.
	if strings.HasPrefix(name, "__Host-") {
		if !c.Secure {
			slog.Warn("web: __Host- cookie requires Secure; forcing Secure=true", "name", name)
			c.Secure = true
		}
		if path != "/" {
			slog.Warn("web: __Host- cookie requires Path=/; forcing Path=/", "name", name)
			path = "/"
		}
		if domain != "" {
			slog.Warn("web: __Host- cookie must not have Domain; clearing", "name", name)
			domain = ""
		}
	}

	// Enforce __Secure- prefix constraints.
	if strings.HasPrefix(name, "__Secure-") && !c.Secure {
		slog.Warn("web: __Secure- cookie requires Secure; forcing Secure=true", "name", name)
		c.Secure = true
	}

	// Enforce Partitioned constraints.
	if c.Partitioned && !c.Secure {
		slog.Warn("web: Partitioned cookie requires Secure; forcing Secure=true", "name", name)
		c.Secure = true
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s=%s", name, quoteValue(value))

	if path != "" {
		fmt.Fprintf(&b, "; Path=%s", path)
	}
	if domain != "" {
		fmt.Fprintf(&b, "; Domain=%s", domain)
	}
	switch {
	case c.MaxAge > 0:
		fmt.Fprintf(&b, "; Max-Age=%d", c.MaxAge)
	case c.MaxAge < 0:
		b.WriteString("; Max-Age=0")
		// MaxAge == 0: omit (session cookie)
	}
	if !c.Expires.IsZero() {
		fmt.Fprintf(&b, "; Expires=%s", c.Expires.UTC().Format(http.TimeFormat))
	}
	if c.Secure {
		b.WriteString("; Secure")
	}
	if c.HTTPOnly {
		b.WriteString("; HttpOnly")
	}
	if sameSite, ok := cookieSameSiteAttr[c.SameSite]; ok {
		fmt.Fprintf(&b, "; SameSite=%s", sameSite)
	} else {
		// Unknown value — emit as-is.
		fmt.Fprintf(&b, "; SameSite=%s", c.SameSite)
	}
	if c.Partitioned {
		b.WriteString("; Partitioned")
	}
	if c.Priority != "" {
		fmt.Fprintf(&b, "; Priority=%s", c.Priority)
	}

	result := b.String()
	if len(result) > 4096 {
		slog.Warn("web: serialized cookie exceeds 4096 bytes; browsers may drop it",
			"name", name, "size", len(result))
	}
	return result
}

// ParseCookies parses a Cookie header value into a slice of cookies.
// Each pair is separated by "; " per RFC 6265.
func ParseCookies(header string) []Cookie {
	if header == "" {
		return nil
	}
	var cookies []Cookie
	pairs := strings.SplitSeq(header, ";")
	for pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		name, value, _ := strings.Cut(pair, "=")
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if name == "" {
			continue
		}
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}
		cookies = append(cookies, Cookie{Name: name, Value: value})
	}
	return cookies
}

// SetCookie appends a Set-Cookie header to the response.
func SetCookie(resp *Response, cookie Cookie) {
	resp.Headers.Append("Set-Cookie", cookie.String())
}

// GetCookie returns the named cookie from the request's Cookie header.
// Returns the cookie and true if found, or a zero Cookie and false otherwise.
func GetCookie(req Request, name string) (Cookie, bool) {
	cookies := ParseCookies(req.Headers.Get("Cookie"))
	for _, c := range cookies {
		if c.Name == name {
			return c, true
		}
	}
	return Cookie{}, false
}
