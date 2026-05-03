package authweb

import (
	"errors"

	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/web"
)

// mapServiceError maps domain error sentinels to transport-facing *web.Error types.
func mapServiceError(err error) error {
	switch {
	case errors.Is(err, auth.ErrUserNotFound):
		return web.ErrNotFound("User not found")
	case errors.Is(err, auth.ErrEmailTaken):
		return web.ErrConflict("That email address is already in use")
	case errors.Is(err, auth.ErrInvalidCredentials):
		return web.ErrUnauthorized("Invalid credentials")
	case errors.Is(err, auth.ErrInvalidVerificationCode):
		return web.ErrValidation("The code you entered is incorrect")
	case errors.Is(err, auth.ErrVerificationExpired):
		return web.ErrValidation("This verification has expired. Please request a new code.")
	case errors.Is(err, auth.ErrTooManyAttempts):
		return web.ErrForbidden("Too many failed attempts. Please request a new code.")
	case errors.Is(err, auth.ErrTokenConsumed):
		return web.ErrForbidden("This verification link has already been used. Please request a new code.")
	case errors.Is(err, auth.ErrVerificationMissing):
		return web.ErrBadRequest("Verification data is missing or invalid")
	case errors.Is(err, auth.ErrAdminExists):
		return web.ErrConflict("An admin account already exists")
	case errors.Is(err, auth.ErrLastAdmin):
		return web.ErrConflict("Cannot remove the last admin account")
	case errors.Is(err, auth.ErrUnsupportedMethod):
		return web.ErrBadRequest("Auth method not supported")
	case errors.Is(err, auth.ErrPasskeyNotFound):
		return web.ErrNotFound("Passkey not found")
	case errors.Is(err, auth.ErrPasskeyAlreadyRegistered):
		return web.ErrConflict("This passkey is already registered")
	case errors.Is(err, auth.ErrPasskeyVerificationFailed):
		return web.ErrUnauthorized("Passkey verification failed")
	case errors.Is(err, auth.ErrPasskeyCloneDetected):
		return web.ErrForbidden("Authenticator may be cloned")
	case errors.Is(err, auth.ErrPasskeyServerState):
		return web.ErrInternal(auth.ErrPasskeyServerState)
	default:
		return web.ErrInternal(err)
	}
}
