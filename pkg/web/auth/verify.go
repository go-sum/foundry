package auth

import (
	"crypto/subtle"
	"errors"
)

var (
	ErrStateMismatch = errors.New("oauth: state parameter mismatch")
	ErrNonceMismatch = errors.New("oauth: nonce mismatch")
)

// VerifyState performs a constant-time comparison of the returned state
// against the expected state from the OAuthTransaction.
func VerifyState(returned, expected string) error {
	if subtle.ConstantTimeCompare([]byte(returned), []byte(expected)) != 1 {
		return ErrStateMismatch
	}
	return nil
}

// VerifyNonce checks that the returned nonce matches the expected nonce.
// If expected is empty, the check is skipped (nonce is optional for non-OIDC flows).
func VerifyNonce(returned, expected string) error {
	if expected == "" {
		return nil
	}
	if subtle.ConstantTimeCompare([]byte(returned), []byte(expected)) != 1 {
		return ErrNonceMismatch
	}
	return nil
}
