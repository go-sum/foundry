package secure

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const signedURLParam = "_sig"

// SignURL appends a cryptographic signature query parameter to rawPath, producing
// a time-limited signed URL. The signature covers the full path (including
// existing query parameters) so that any modification to the URL invalidates it.
//
// key must be at least 32 bytes. ttl must be positive.
//
// Example:
//
//	signed, err := secure.SignURL("/downloads/report.pdf", key, 24*time.Hour)
//	// signed == "/downloads/report.pdf?_sig=<token>"
func SignURL(rawPath string, key []byte, ttl time.Duration) (string, error) {
	if len(key) < 32 {
		return "", ErrKeyTooShort
	}
	canonical := stripSigParam(rawPath)
	token, err := IssueToken(key, canonical, ttl)
	if err != nil {
		return "", fmt.Errorf("secure: sign URL: %w", err)
	}
	if containsQuery(canonical) {
		return canonical + "&" + signedURLParam + "=" + token, nil
	}
	return canonical + "?" + signedURLParam + "=" + token, nil
}

// VerifyURL checks the signature of a signed URL produced by SignURL.
// On success it returns the original path (without the _sig parameter).
// Returns ErrTokenInvalid or ErrTokenExpired on failure.
func VerifyURL(rawPath string, key []byte) (string, error) {
	canonical := stripSigParam(rawPath)
	sig := extractSigParam(rawPath)
	if sig == "" {
		return "", ErrTokenInvalid
	}
	if err := VerifyToken(key, canonical, sig); err != nil {
		return "", err
	}
	return canonical, nil
}

// stripSigParam removes the _sig query parameter from rawPath, leaving
// all other query parameters intact.
func stripSigParam(rawPath string) string {
	u, err := url.Parse(rawPath)
	if err != nil {
		return rawPath
	}
	q := u.Query()
	q.Del(signedURLParam)
	u.RawQuery = q.Encode()
	if u.RawQuery == "" {
		return u.Path
	}
	return u.Path + "?" + u.RawQuery
}

// extractSigParam returns the value of the _sig query parameter from rawPath.
func extractSigParam(rawPath string) string {
	u, err := url.Parse(rawPath)
	if err != nil {
		return ""
	}
	return u.Query().Get(signedURLParam)
}

// containsQuery reports whether rawPath has a query string component.
func containsQuery(rawPath string) bool {
	_, after, ok := strings.Cut(rawPath, "?")
	return ok && after != ""
}
