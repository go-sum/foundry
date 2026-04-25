package auth

import "errors"

var (
	ErrUserNotFound              = errors.New("auth: user not found")
	ErrEmailTaken                = errors.New("auth: email already in use")
	ErrAdminExists               = errors.New("auth: admin already exists")
	ErrInvalidCredentials        = errors.New("auth: invalid credentials")
	ErrInvalidVerificationCode   = errors.New("auth: invalid verification code")
	ErrVerificationExpired       = errors.New("auth: verification expired")
	ErrVerificationMissing       = errors.New("auth: verification missing")
	ErrUnsupportedMethod         = errors.New("auth: unsupported auth method")
	ErrWebAuthnIDAlreadySet      = errors.New("auth: webauthn id already set")
	ErrPasskeyNotFound           = errors.New("auth: passkey not found")
	ErrPasskeyAlreadyRegistered  = errors.New("auth: passkey already registered")
	ErrPasskeyVerificationFailed = errors.New("auth: passkey verification failed")
	ErrPasskeyCloneDetected      = errors.New("auth: passkey clone detected")
	ErrPasskeyServerState        = errors.New("auth: passkey server state error")
)
