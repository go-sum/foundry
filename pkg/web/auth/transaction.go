package auth

import "time"

// SessionKey is the session key under which an OAuthTransaction is stored.
const SessionKey = "oauth.transaction"

// OAuthTransaction carries cross-request OAuth state stored in the session.
// Store it under SessionKey with session.Set, retrieve with session.Get.
type OAuthTransaction struct {
	State     string    // random CSRF state parameter
	Nonce     string    // random OpenID Connect nonce (optional; empty if not used)
	Verifier  string    // RFC 7636 PKCE code_verifier for the token exchange
	ReturnTo  string    // URL to redirect to after successful auth (sanitized)
	CreatedAt time.Time // when the transaction was created (UTC)
}

// NewTransaction creates a new OAuthTransaction with freshly generated
// State, Nonce, and Verifier. ReturnTo is sanitized via SanitizeReturnTo;
// if returnTo is invalid, ReturnTo is set to "/".
func NewTransaction(returnTo string) (OAuthTransaction, error) {
	state, err := NewVerifier()
	if err != nil {
		return OAuthTransaction{}, err
	}
	nonce, err := NewVerifier()
	if err != nil {
		return OAuthTransaction{}, err
	}
	verifier, err := NewVerifier()
	if err != nil {
		return OAuthTransaction{}, err
	}
	return OAuthTransaction{
		State:     state,
		Nonce:     nonce,
		Verifier:  verifier,
		ReturnTo:  SanitizeReturnTo(returnTo),
		CreatedAt: time.Now().UTC(),
	}, nil
}
