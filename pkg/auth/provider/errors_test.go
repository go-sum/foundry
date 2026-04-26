package provider

import (
	"testing"
)

// TestErrorSentinelsUnique verifies that no two error sentinels share the same
// message string. Duplicate messages would make errors.Is lookups ambiguous
// and break error log classification.
func TestErrorSentinelsUnique(t *testing.T) {
	sentinels := map[string]error{
		"ErrClientNotFound":       ErrClientNotFound,
		"ErrInvalidRedirectURI":   ErrInvalidRedirectURI,
		"ErrInvalidScope":         ErrInvalidScope,
		"ErrCodeExpired":          ErrCodeExpired,
		"ErrCodeUsed":             ErrCodeUsed,
		"ErrCodeNotFound":         ErrCodeNotFound,
		"ErrPKCERequired":         ErrPKCERequired,
		"ErrPKCEFailed":           ErrPKCEFailed,
		"ErrTokenRevoked":         ErrTokenRevoked,
		"ErrTokenExpired":         ErrTokenExpired,
		"ErrTokenNotFound":        ErrTokenNotFound,
		"ErrInvalidGrant":         ErrInvalidGrant,
		"ErrUnsupportedGrantType": ErrUnsupportedGrantType,
		"ErrMissingChallenge":     ErrMissingChallenge,
		"ErrConsentNotFound":      ErrConsentNotFound,
	}

	seen := make(map[string]string) // message → sentinel name
	for name, err := range sentinels {
		msg := err.Error()
		if prior, exists := seen[msg]; exists {
			t.Errorf("duplicate error message %q shared by %s and %s", msg, prior, name)
		}
		seen[msg] = name
	}
}

// TestErrorSentinelsNonNil verifies that none of the exported sentinels is nil.
func TestErrorSentinelsNonNil(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"ErrClientNotFound", ErrClientNotFound},
		{"ErrInvalidRedirectURI", ErrInvalidRedirectURI},
		{"ErrInvalidScope", ErrInvalidScope},
		{"ErrCodeExpired", ErrCodeExpired},
		{"ErrCodeUsed", ErrCodeUsed},
		{"ErrCodeNotFound", ErrCodeNotFound},
		{"ErrPKCERequired", ErrPKCERequired},
		{"ErrPKCEFailed", ErrPKCEFailed},
		{"ErrTokenRevoked", ErrTokenRevoked},
		{"ErrTokenExpired", ErrTokenExpired},
		{"ErrTokenNotFound", ErrTokenNotFound},
		{"ErrInvalidGrant", ErrInvalidGrant},
		{"ErrUnsupportedGrantType", ErrUnsupportedGrantType},
		{"ErrMissingChallenge", ErrMissingChallenge},
		{"ErrConsentNotFound", ErrConsentNotFound},
	}
	for _, tc := range cases {
		if tc.err == nil {
			t.Errorf("%s is nil", tc.name)
		}
	}
}
