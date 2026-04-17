package auth

import "strings"

// SanitizeReturnTo validates returnTo and returns it if safe, otherwise "/".
// A safe return URL must:
//   - Start with "/" (relative URL)
//   - NOT start with "//" (protocol-relative — open redirect vector)
//   - NOT contain a newline or carriage return (header injection)
func SanitizeReturnTo(returnTo string) string {
	if !strings.HasPrefix(returnTo, "/") {
		return "/"
	}
	if strings.HasPrefix(returnTo, "//") {
		return "/"
	}
	if strings.ContainsAny(returnTo, "\r\n") {
		return "/"
	}
	return returnTo
}
