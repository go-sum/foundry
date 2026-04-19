package config

import "errors"

var (
	// ErrCSRFKeyMissing is returned when SECURITY_CSRF_KEY is not set.
	ErrCSRFKeyMissing = errors.New("config: SECURITY_CSRF_KEY is required")

	// ErrCSRFKeyInvalid is returned when SECURITY_CSRF_KEY is not valid key material.
	ErrCSRFKeyInvalid = errors.New("config: SECURITY_CSRF_KEY must be valid key material")

	// ErrCSRFPrevKeysInvalid is returned when the csrf previous keys are invalid.
	ErrCSRFPrevKeysInvalid = errors.New("config: csrf previous keys must be valid")

	// ErrSessionKeyMissing is returned when SESSION_STORE=cookie but SECURITY_SESSION_KEY is not set.
	ErrSessionKeyMissing = errors.New("config: SECURITY_SESSION_KEY is required for cookie session store")

	// ErrSessionKeyInvalid is returned when SECURITY_SESSION_KEY is not valid hex-encoded key material.
	ErrSessionKeyInvalid = errors.New("config: SECURITY_SESSION_KEY must be valid hex-encoded key material")
)
