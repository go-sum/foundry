package headers

import "strings"

// ParseParams parses a semicolon-separated list of key=value parameters,
// where values may be optionally double-quoted with backslash escape support.
// Keys are lowercased. Returns nil if input is empty.
// Example: `charset="utf-8"; boundary=--abc` → {"charset":"utf-8","boundary":"--abc"}
func ParseParams(input string) map[string]string {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	result := make(map[string]string)
	parts := strings.Split(input, ";")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		idx := strings.IndexByte(part, '=')
		if idx < 0 {
			// Boolean token with no value — skip for params map; callers handle these
			continue
		}

		k := strings.ToLower(strings.TrimSpace(part[:idx]))
		v := strings.TrimSpace(part[idx+1:])

		if k == "" {
			continue
		}

		if len(v) >= 2 && v[0] == '"' {
			v = decodeQuotedString(v)
		}

		result[k] = v
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// decodeQuotedString decodes a double-quoted string per RFC 7230 3.2.6.
// The input must start with '"'. Handles \" and \\ escape sequences.
func decodeQuotedString(s string) string {
	if len(s) < 2 || s[0] != '"' {
		return s
	}

	var b strings.Builder
	i := 1
	for i < len(s) {
		c := s[i]
		if c == '"' {
			break
		}
		if c == '\\' && i+1 < len(s) {
			i++
			b.WriteByte(s[i])
			i++
			continue
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

// Quote returns v wrapped in double quotes if it contains chars that require
// quoting in header parameter values (spaces, semicolons, commas, double quotes,
// equals signs, or backslashes). Otherwise returns v unchanged.
func Quote(v string) string {
	for _, c := range v {
		switch c {
		case ' ', ';', ',', '"', '=', '\\':
			return `"` + strings.ReplaceAll(strings.ReplaceAll(v, `\`, `\\`), `"`, `\"`) + `"`
		}
	}
	return v
}
