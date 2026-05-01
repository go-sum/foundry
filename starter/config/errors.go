package config

import "errors"

var (
	// ErrCSRFKeyMissing is returned when SECURITY_CSRF_KEY is not set.
	ErrCSRFKeyMissing = errors.New("config: SECURITY_CSRF_KEY is required")

	// ErrCSRFKeyInvalid is returned when SECURITY_CSRF_KEY is not valid key material.
	ErrCSRFKeyInvalid = errors.New("config: SECURITY_CSRF_KEY must be valid key material")

	// ErrSessionKeyMissing is returned when SESSION_STORE=cookie but SECURITY_SESSION_KEY is not set.
	ErrSessionKeyMissing = errors.New("config: SECURITY_SESSION_KEY is required for cookie session store")

	// ErrSessionKeyInvalid is returned when SECURITY_SESSION_KEY is not valid hex-encoded key material.
	ErrSessionKeyInvalid = errors.New("config: SECURITY_SESSION_KEY must be valid hex-encoded key material")

	// ErrSessionStoreMemoryTestingOnly is returned when the in-memory session
	// store is selected outside the test environment.
	ErrSessionStoreMemoryTestingOnly = errors.New("config: SESSION_STORE=memory is only allowed when APP_ENV=testing")

	// ErrKVStoreUnavailable is returned when a selected KV-backed feature cannot
	// reach the configured KV service at startup.
	ErrKVStoreUnavailable = errors.New("config: KV store is required but unavailable")

	// ErrKVPasswordMissing is returned when SESSION_STORE=kv is selected in a
	// non-test environment without a password in KV_URL.
	ErrKVPasswordMissing = errors.New("config: KV_URL must include a password for kv session store outside testing")

	// ErrKVSessionStoreUnsupported is returned when SESSION_STORE=kv is selected
	// with a KV implementation that does not provide session primitives.
	ErrKVSessionStoreUnsupported = errors.New("config: SESSION_STORE=kv requires a session-capable KV store")

	// ErrAuthTokenKeyMissing is returned when SECURITY_AUTH_TOKEN_KEY is not set.
	ErrAuthTokenKeyMissing = errors.New("config: SECURITY_AUTH_TOKEN_KEY is required")

	// ErrAuthTokenKeyInvalid is returned when SECURITY_AUTH_TOKEN_KEY is not valid hex-encoded key material.
	ErrAuthTokenKeyInvalid = errors.New("config: SECURITY_AUTH_TOKEN_KEY must be valid hex-encoded key material (minimum 32 bytes)")

	// ErrBaseURLMissing is returned when SITE_BASE_URL is not set in production.
	ErrBaseURLMissing = errors.New("config: SITE_BASE_URL is required in production")

	// ErrAllowedHostsEmpty is returned when AllowedHosts is empty in production,
	// which would silently disable host-header validation.
	ErrAllowedHostsEmpty = errors.New("config: AllowedHosts must not be empty in production — set SITE_BASE_URL or SITE_ALLOWED_HOSTS")

	// ErrRateLimitStoreUnsupported is returned when the shared KV store does not
	// provide the rate-limit-specific backend methods required by pkg/web/ratelimit.
	ErrRateLimitStoreUnsupported = errors.New("config: shared KV store does not support rate limiting")
)
