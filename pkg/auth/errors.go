package auth

import "errors"

var (
	ErrTokenKeyMissing           = errors.New("auth: token key is required")
	ErrTokenKeyInvalid           = errors.New("auth: token key is invalid (must be 32+ bytes of hex)")
	ErrUserNotFound              = errors.New("auth: user not found")
	ErrEmailTaken                = errors.New("auth: email already in use")
	ErrAdminExists               = errors.New("auth: admin already exists")
	ErrLastAdmin                 = errors.New("auth: cannot remove the last admin")
	ErrInvalidCredentials        = errors.New("auth: invalid credentials")
	ErrInvalidVerificationCode   = errors.New("auth: invalid verification code")
	ErrVerificationExpired       = errors.New("auth: verification expired")
	ErrVerificationMissing       = errors.New("auth: verification missing")
	ErrTooManyAttempts           = errors.New("auth: too many verification attempts")
	ErrTokenConsumed             = errors.New("auth: token already consumed")
	ErrUnsupportedMethod         = errors.New("auth: unsupported auth method")
	ErrWebAuthnIDAlreadySet      = errors.New("auth: webauthn id already set")
	ErrPasskeyNotFound           = errors.New("auth: passkey not found")
	ErrPasskeyAlreadyRegistered  = errors.New("auth: passkey already registered")
	ErrPasskeyVerificationFailed = errors.New("auth: passkey verification failed")
	ErrPasskeyCloneDetected      = errors.New("auth: passkey clone detected")
	ErrPasskeyServerState        = errors.New("auth: passkey server state error")
)
