package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UserReader provides read-only access to user records.
type UserReader interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
}

// UserWriter provides read-write access to user records.
type UserWriter interface {
	UserReader
	CreateUser(ctx context.Context, email, displayName string, role Role, verified bool) (User, error)
	UpdateUserEmail(ctx context.Context, id uuid.UUID, email string) (User, error)
	SetWebAuthnID(ctx context.Context, id uuid.UUID, webauthnID []byte) (User, error)
	SetWebAuthnIDIfNull(ctx context.Context, id uuid.UUID, webauthnID []byte) (User, error)
	GetUserByWebAuthnID(ctx context.Context, webauthnID []byte) (User, error)
}

// Notifier delivers verification messages to end users.
type Notifier interface {
	SendVerification(ctx context.Context, input DeliveryInput) error
}

// maxVerifyAttempts is the number of failed TOTP attempts allowed before the
// pending flow is locked. Requires requesting a new code after this limit.
const maxVerifyAttempts = 5

// AuthService implements email-TOTP authentication flows.
type AuthService struct {
	users      UserWriter
	notifier   Notifier
	tokenCodec TokenCodec
	nonceStore TokenNonceStore
	config     EmailTOTPConfig
	clock      func() time.Time
}

// AuthServiceConfig configures an AuthService.
type AuthServiceConfig struct {
	Users       UserWriter
	Notifier    Notifier
	TokenCodec  TokenCodec
	NonceStore  TokenNonceStore
	EmailTOTP   EmailTOTPConfig
	Clock       func() time.Time
}

// NewAuthService returns a new AuthService.
func NewAuthService(cfg AuthServiceConfig) *AuthService {
	clock := cfg.Clock
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	notifier := cfg.Notifier
	if notifier == nil {
		notifier = noopNotifier{}
	}
	return &AuthService{
		users:      cfg.Users,
		notifier:   notifier,
		tokenCodec: cfg.TokenCodec,
		nonceStore: cfg.NonceStore,
		config:     cfg.EmailTOTP,
		clock:      clock,
	}
}

// BeginSignup starts a signup verification flow for a new user.
// Anti-enumeration: when the email is already registered and verified, a
// FlowAlreadyRegistered notification is sent (no code or link) and a dummy
// flow is returned so the response is indistinguishable from a real signup.
func (s *AuthService) BeginSignup(ctx context.Context, input BeginSignupInput, verifyPath string) (PendingFlow, error) {
	if !s.config.Enabled {
		return PendingFlow{}, ErrUnsupportedMethod
	}

	existing, lookupErr := s.users.GetUserByEmail(ctx, input.Email)
	switch {
	case lookupErr == nil && existing.Verified:
		_ = s.notifier.SendVerification(ctx, DeliveryInput{
			Purpose: FlowAlreadyRegistered,
			Email:   input.Email,
		})
		flow, _, err := s.newPendingFlow(FlowSignup, input.Email, input.DisplayName, RoleUser, uuid.Nil)
		if err != nil {
			return PendingFlow{}, err
		}
		return flow, nil
	case lookupErr != nil && !errors.Is(lookupErr, ErrUserNotFound):
		return PendingFlow{}, fmt.Errorf("lookup signup email: %w", lookupErr)
	}
	// ErrUserNotFound (new user) or exists-but-unverified: proceed normally.

	flow, code, err := s.newPendingFlow(FlowSignup, input.Email, input.DisplayName, RoleUser, uuid.Nil)
	if err != nil {
		return PendingFlow{}, err
	}
	if err := s.deliver(ctx, flow, code, verifyPath); err != nil {
		return PendingFlow{}, err
	}
	return flow, nil
}

// BeginSignin starts a signin verification flow. Anti-enumeration: always
// returns a PendingFlow regardless of whether the user exists.
func (s *AuthService) BeginSignin(ctx context.Context, input BeginSigninInput, verifyPath string) (PendingFlow, error) {
	if !s.config.Enabled {
		return PendingFlow{}, ErrUnsupportedMethod
	}

	flow, code, err := s.newPendingFlow(FlowSignin, input.Email, "", "", uuid.Nil)
	if err != nil {
		return PendingFlow{}, err
	}

	user, err := s.users.GetUserByEmail(ctx, input.Email)
	switch {
	case err == nil && user.Verified:
		if err := s.deliver(ctx, flow, code, verifyPath); err != nil {
			return PendingFlow{}, err
		}
	case err == nil:
		slog.DebugContext(ctx, "auth: signin suppressed (anti-enumeration)")
	case errors.Is(err, ErrUserNotFound):
		slog.DebugContext(ctx, "auth: signin suppressed (anti-enumeration)")
	case err != nil:
		return PendingFlow{}, fmt.Errorf("lookup signin email: %w", err)
	}

	return flow, nil
}

// BeginEmailChange starts an email change verification flow for an existing user.
func (s *AuthService) BeginEmailChange(ctx context.Context, userID uuid.UUID, input BeginEmailChangeInput, verifyPath string) (PendingFlow, error) {
	if !s.config.Enabled {
		return PendingFlow{}, ErrUnsupportedMethod
	}

	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return PendingFlow{}, fmt.Errorf("lookup current user: %w", err)
	}

	if strings.EqualFold(user.Email, input.Email) {
		return PendingFlow{}, ErrEmailTaken
	}

	if _, err := s.users.GetUserByEmail(ctx, input.Email); err == nil {
		return PendingFlow{}, ErrEmailTaken
	} else if !errors.Is(err, ErrUserNotFound) {
		return PendingFlow{}, fmt.Errorf("lookup email change target: %w", err)
	}

	flow, code, err := s.newPendingFlow(FlowEmailChange, input.Email, "", "", userID)
	if err != nil {
		return PendingFlow{}, err
	}
	if err := s.deliver(ctx, flow, code, verifyPath); err != nil {
		return PendingFlow{}, err
	}
	return flow, nil
}

// ResendPendingFlow re-initiates the pending verification flow, generating a
// new secret and code.
func (s *AuthService) ResendPendingFlow(ctx context.Context, flow PendingFlow, verifyPath string) (PendingFlow, error) {
	switch flow.Purpose {
	case FlowSignup:
		return s.BeginSignup(ctx, BeginSignupInput{
			Email:       flow.Email,
			DisplayName: flow.DisplayName,
		}, verifyPath)
	case FlowSignin:
		return s.BeginSignin(ctx, BeginSigninInput{
			Email: flow.Email,
		}, verifyPath)
	case FlowEmailChange:
		return s.BeginEmailChange(ctx, flow.UserID, BeginEmailChangeInput{
			Email: flow.Email,
		}, verifyPath)
	default:
		return PendingFlow{}, ErrUnsupportedMethod
	}
}

// VerifyPendingFlow verifies the code against the session-held pending flow.
// It returns the updated flow (with incremented Attempts) so the caller can
// persist it back to the session on failure.
func (s *AuthService) VerifyPendingFlow(ctx context.Context, flow PendingFlow, input VerifyInput) (VerifyResult, PendingFlow, error) {
	if flow.Attempts >= maxVerifyAttempts {
		return VerifyResult{}, flow, ErrTooManyAttempts
	}
	flow.Attempts++
	if err := validateTOTPCode(flow.Secret, flow.IssuedAt, flow.ExpiresAt, input.Code, s.config.PeriodSeconds, s.clock()); err != nil {
		return VerifyResult{}, flow, err
	}
	result, err := s.finishVerification(ctx, flow)
	return result, flow, err
}

// VerifyToken verifies the code against a self-contained token from a verify link.
func (s *AuthService) VerifyToken(ctx context.Context, token string, input VerifyInput) (VerifyResult, error) {
	payload, err := s.tokenCodec.Decode(token)
	if err != nil {
		return VerifyResult{}, err
	}

	nonceKey := tokenNonceKey(token)
	if s.nonceStore != nil {
		consumed, err := s.nonceStore.HasConsumed(ctx, nonceKey)
		if err != nil {
			return VerifyResult{}, fmt.Errorf("check token nonce: %w", err)
		}
		if consumed {
			return VerifyResult{}, ErrTokenConsumed
		}
	}

	if err := validateTOTPCode(payload.Secret, payload.IssuedAt, payload.ExpiresAt, input.Code, s.config.PeriodSeconds, s.clock()); err != nil {
		return VerifyResult{}, err
	}

	if s.nonceStore != nil {
		ttl := payload.ExpiresAt.Sub(s.clock())
		if ttl > 0 {
			if err := s.nonceStore.MarkConsumed(ctx, nonceKey, ttl); err != nil {
				return VerifyResult{}, fmt.Errorf("mark token consumed: %w", err)
			}
		}
	}

	return s.finishVerification(ctx, pendingFlowFromToken(payload))
}

// VerifyPageState decodes a verify token and returns the display context
// needed to render the verification page.
func (s *AuthService) VerifyPageState(token string) (VerifyPageState, error) {
	payload, err := s.tokenCodec.Decode(token)
	if err != nil {
		return VerifyPageState{}, err
	}
	return VerifyPageState{
		Purpose: payload.Purpose,
		Token:   token,
		Email:   payload.Email,
	}, nil
}

func (s *AuthService) newPendingFlow(purpose FlowPurpose, email, displayName string, role Role, userID uuid.UUID) (PendingFlow, string, error) {
	now := s.clock()
	secret, err := randomSecret()
	if err != nil {
		return PendingFlow{}, "", fmt.Errorf("generate verification secret: %w", err)
	}

	period := time.Duration(s.config.PeriodSeconds) * time.Second
	if period <= 0 {
		period = 5 * time.Minute
	}

	flow := PendingFlow{
		Purpose:     purpose,
		Email:       email,
		DisplayName: displayName,
		Role:        role,
		UserID:      userID,
		Secret:      secret,
		IssuedAt:    now,
		ExpiresAt:   now.Add(period),
	}

	code, err := generateTOTPCode(secret, now, s.config.PeriodSeconds)
	if err != nil {
		return PendingFlow{}, "", err
	}

	return flow, code, nil
}

func (s *AuthService) deliver(ctx context.Context, flow PendingFlow, code, verifyPath string) error {
	token, err := s.tokenCodec.Encode(verificationTokenFromFlow(flow))
	if err != nil {
		return fmt.Errorf("encode verification token: %w", err)
	}

	verifyURL, err := appendVerifyToken(verifyPath, token)
	if err != nil {
		return fmt.Errorf("build verify url: %w", err)
	}

	if err := s.notifier.SendVerification(ctx, DeliveryInput{
		Purpose:   flow.Purpose,
		Email:     flow.Email,
		Code:      code,
		VerifyURL: verifyURL,
		ExpiresAt: flow.ExpiresAt,
	}); err != nil {
		return fmt.Errorf("send verification email: %w", err)
	}

	return nil
}

func (s *AuthService) finishVerification(ctx context.Context, flow PendingFlow) (VerifyResult, error) {
	switch flow.Purpose {
	case FlowSignup:
		return s.finishSignup(ctx, flow)
	case FlowSignin:
		return s.finishSignin(ctx, flow)
	case FlowEmailChange:
		return s.finishEmailChange(ctx, flow)
	default:
		return VerifyResult{}, ErrUnsupportedMethod
	}
}

func (s *AuthService) finishSignup(ctx context.Context, flow PendingFlow) (VerifyResult, error) {
	user, err := s.users.CreateUser(ctx, flow.Email, flow.DisplayName, RoleUser, true)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			existing, getErr := s.users.GetUserByEmail(ctx, flow.Email)
			if getErr == nil && existing.Verified {
				return VerifyResult{Purpose: FlowSignup, User: existing, Method: string(MethodEmailTOTP)}, nil
			}
		}
		return VerifyResult{}, fmt.Errorf("create verified user: %w", err)
	}
	return VerifyResult{Purpose: FlowSignup, User: user, Method: string(MethodEmailTOTP)}, nil
}

func (s *AuthService) finishSignin(ctx context.Context, flow PendingFlow) (VerifyResult, error) {
	user, err := s.users.GetUserByEmail(ctx, flow.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return VerifyResult{}, ErrInvalidCredentials
		}
		return VerifyResult{}, fmt.Errorf("lookup signin user: %w", err)
	}
	if !user.Verified {
		return VerifyResult{}, ErrInvalidCredentials
	}
	return VerifyResult{Purpose: FlowSignin, User: user, Method: string(MethodEmailTOTP)}, nil
}

func (s *AuthService) finishEmailChange(ctx context.Context, flow PendingFlow) (VerifyResult, error) {
	user, err := s.users.GetUserByID(ctx, flow.UserID)
	if err != nil {
		return VerifyResult{}, fmt.Errorf("lookup email-change user: %w", err)
	}
	if strings.EqualFold(user.Email, flow.Email) {
		return VerifyResult{Purpose: FlowEmailChange, User: user, Method: string(MethodEmailTOTP)}, nil
	}

	user, err = s.users.UpdateUserEmail(ctx, flow.UserID, flow.Email)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			current, getErr := s.users.GetUserByID(ctx, flow.UserID)
			if getErr == nil && strings.EqualFold(current.Email, flow.Email) {
				return VerifyResult{Purpose: FlowEmailChange, User: current, Method: string(MethodEmailTOTP)}, nil
			}
		}
		return VerifyResult{}, fmt.Errorf("update verified email: %w", err)
	}
	return VerifyResult{Purpose: FlowEmailChange, User: user, Method: string(MethodEmailTOTP)}, nil
}

func appendVerifyToken(basePath, token string) (string, error) {
	u, err := url.Parse(basePath)
	if err != nil {
		return "", err
	}
	query := u.Query()
	query.Set("token", token)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

type noopNotifier struct{}

func (noopNotifier) SendVerification(context.Context, DeliveryInput) error {
	return errors.New("verification notifier required")
}
