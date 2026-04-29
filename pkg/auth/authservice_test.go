package auth

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ─── Test doubles ─────────────────────────────────────────────────────────────

type fixedTokenCodec struct {
	token VerificationToken
}

func (f fixedTokenCodec) Encode(VerificationToken) (string, error) { return "fixed-token", nil }
func (f fixedTokenCodec) Decode(string) (VerificationToken, error)  { return f.token, nil }

type failingTokenCodec struct{ err error }

func (f failingTokenCodec) Encode(VerificationToken) (string, error) { return "", f.err }
func (f failingTokenCodec) Decode(string) (VerificationToken, error)  { return VerificationToken{}, f.err }

// fakeUserStore implements UserWriter for testing.
type fakeUserStore struct {
	getUserByEmailResult  User
	getUserByEmailErr     error
	getUserByIDResult     User
	getUserByIDErr        error
	getUserByIDFn         func(uuid.UUID) (User, error) // overrides result/err when set
	createUserResult      User
	createUserErr         error
	updateUserEmailResult User
	updateUserEmailErr    error
}

func (f *fakeUserStore) GetUserByEmail(_ context.Context, _ string) (User, error) {
	return f.getUserByEmailResult, f.getUserByEmailErr
}
func (f *fakeUserStore) GetUserByID(_ context.Context, id uuid.UUID) (User, error) {
	if f.getUserByIDFn != nil {
		return f.getUserByIDFn(id)
	}
	return f.getUserByIDResult, f.getUserByIDErr
}
func (f *fakeUserStore) CreateUser(_ context.Context, _, _ string, _ Role, _ bool) (User, error) {
	return f.createUserResult, f.createUserErr
}
func (f *fakeUserStore) UpdateUserEmail(_ context.Context, _ uuid.UUID, _ string) (User, error) {
	return f.updateUserEmailResult, f.updateUserEmailErr
}
func (f *fakeUserStore) SetWebAuthnID(_ context.Context, _ uuid.UUID, _ []byte) (User, error) {
	return User{}, nil
}
func (f *fakeUserStore) SetWebAuthnIDIfNull(_ context.Context, _ uuid.UUID, _ []byte) (User, error) {
	return User{}, nil
}
func (f *fakeUserStore) GetUserByWebAuthnID(_ context.Context, _ []byte) (User, error) {
	return User{}, nil
}

// fakeNotifier records calls and captures the delivery input; injects err when set.
type fakeNotifier struct {
	called bool
	input  DeliveryInput
	err    error
}

func (f *fakeNotifier) SendVerification(_ context.Context, input DeliveryInput) error {
	f.called = true
	f.input = input
	return f.err
}

// fakeNonceStore is an in-memory TokenNonceStore with optional error injection.
type fakeNonceStore struct {
	consumed        map[string]struct{}
	hasConsumedErr  error
	markConsumedErr error
}

func newFakeNonceStore() *fakeNonceStore {
	return &fakeNonceStore{consumed: make(map[string]struct{})}
}

func (f *fakeNonceStore) HasConsumed(_ context.Context, key string) (bool, error) {
	if f.hasConsumedErr != nil {
		return false, f.hasConsumedErr
	}
	_, ok := f.consumed[key]
	return ok, nil
}

func (f *fakeNonceStore) MarkConsumed(_ context.Context, key string, _ time.Duration) error {
	if f.markConsumedErr != nil {
		return f.markConsumedErr
	}
	f.consumed[key] = struct{}{}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// fixedClock is the pinned instant used across authservice tests.
var fixedClock = func() time.Time { return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) }

func newTestAuthService(store *fakeUserStore, notifier *fakeNotifier) *AuthService {
	return NewAuthService(AuthServiceConfig{
		Users:      store,
		Notifier:   notifier,
		TokenCodec: stubTokenCodec{},
		NonceStore: newFakeNonceStore(),
		EmailTOTP:  EmailTOTPConfig{Enabled: true, PeriodSeconds: 300},
		Clock:      fixedClock,
	})
}

// validFlow returns a PendingFlow and matching TOTP code locked to fixedClock.
func validFlow(t *testing.T, purpose FlowPurpose, email string, userID uuid.UUID) (PendingFlow, string) {
	t.Helper()
	secret, err := randomSecret()
	if err != nil {
		t.Fatalf("randomSecret() error = %v", err)
	}
	now := fixedClock()
	code, err := generateTOTPCode(secret, now, 300)
	if err != nil {
		t.Fatalf("generateTOTPCode() error = %v", err)
	}
	return PendingFlow{
		Purpose:   purpose,
		Email:     email,
		UserID:    userID,
		Secret:    secret,
		IssuedAt:  now,
		ExpiresAt: now.Add(5 * time.Minute),
	}, code
}

// ─── BeginSignin ──────────────────────────────────────────────────────────────

func TestBeginSignin_VerifiedUser_TriggersDelivery(t *testing.T) {
	store := &fakeUserStore{
		getUserByEmailResult: User{ID: uuid.New(), Email: "user@example.com", Verified: true},
	}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	flow, err := svc.BeginSignin(context.Background(), BeginSigninInput{Email: "user@example.com"}, "/verify")
	if err != nil {
		t.Fatalf("BeginSignin() error = %v", err)
	}
	if flow.Purpose != FlowSignin {
		t.Errorf("flow.Purpose = %q, want %q", flow.Purpose, FlowSignin)
	}
	if !notifier.called {
		t.Error("notifier.SendVerification was not called for verified user")
	}
}

func TestBeginSignin_UnknownUser_ReturnsPendingFlow_NoDelivery(t *testing.T) {
	store := &fakeUserStore{getUserByEmailErr: ErrUserNotFound}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	flow, err := svc.BeginSignin(context.Background(), BeginSigninInput{Email: "nobody@example.com"}, "/verify")
	if err != nil {
		t.Fatalf("BeginSignin() error = %v", err)
	}
	if flow.Purpose != FlowSignin {
		t.Errorf("flow.Purpose = %q, want %q", flow.Purpose, FlowSignin)
	}
	if notifier.called {
		t.Error("notifier.SendVerification was called for unknown user (anti-enumeration violation)")
	}
}

func TestBeginSignin_UnverifiedUser_ReturnsPendingFlow_NoDelivery(t *testing.T) {
	store := &fakeUserStore{
		getUserByEmailResult: User{ID: uuid.New(), Email: "unverified@example.com", Verified: false},
	}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	flow, err := svc.BeginSignin(context.Background(), BeginSigninInput{Email: "unverified@example.com"}, "/verify")
	if err != nil {
		t.Fatalf("BeginSignin() error = %v", err)
	}
	if flow.Purpose != FlowSignin {
		t.Errorf("flow.Purpose = %q, want %q", flow.Purpose, FlowSignin)
	}
	if notifier.called {
		t.Error("notifier.SendVerification was called for unverified user (anti-enumeration violation)")
	}
}

// TestBeginSignin_SuppressedBranches_LogsNoEmail verifies that the anti-enumeration
// log lines for unknown and unverified users do NOT include the submitted email
// address in the log output. These tests must NOT use t.Parallel() because they
// mutate the global slog default logger.
func TestBeginSignin_SuppressedBranches_LogsNoEmail(t *testing.T) {
	const testEmail = "secret@example.com"

	tests := []struct {
		name  string
		store *fakeUserStore
	}{
		{
			name:  "unknown user (ErrUserNotFound)",
			store: &fakeUserStore{getUserByEmailErr: ErrUserNotFound},
		},
		{
			name: "unverified user",
			store: &fakeUserStore{
				getUserByEmailResult: User{ID: uuid.New(), Email: testEmail, Verified: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
			orig := slog.Default()
			slog.SetDefault(slog.New(handler))
			t.Cleanup(func() { slog.SetDefault(orig) })

			notifier := &fakeNotifier{}
			svc := newTestAuthService(tt.store, notifier)

			_, err := svc.BeginSignin(context.Background(), BeginSigninInput{Email: testEmail}, "/verify")
			if err != nil {
				t.Fatalf("BeginSignin() error = %v", err)
			}
			if errors.Is(err, ErrUserNotFound) {
				t.Fatalf("BeginSignin() propagated ErrUserNotFound, want nil")
			}

			if strings.Contains(buf.String(), testEmail) {
				t.Errorf("log output contains submitted email %q (PII leak): %s", testEmail, buf.String())
			}
		})
	}
}

func TestBeginSignin_Disabled_ReturnsErrUnsupportedMethod(t *testing.T) {
	svc := NewAuthService(AuthServiceConfig{
		Users:      &fakeUserStore{},
		TokenCodec: stubTokenCodec{},
		EmailTOTP:  EmailTOTPConfig{Enabled: false},
		Clock:      fixedClock,
	})
	_, err := svc.BeginSignin(context.Background(), BeginSigninInput{Email: "user@example.com"}, "/verify")
	if !errors.Is(err, ErrUnsupportedMethod) {
		t.Errorf("BeginSignin() error = %v, want ErrUnsupportedMethod", err)
	}
}

func TestBeginSignin_DBError_ReturnsError(t *testing.T) {
	dbErr := errors.New("db unavailable")
	store := &fakeUserStore{getUserByEmailErr: dbErr}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	_, err := svc.BeginSignin(context.Background(), BeginSigninInput{Email: "user@example.com"}, "/verify")
	if !errors.Is(err, dbErr) {
		t.Errorf("BeginSignin() error = %v, want wrapped %v", err, dbErr)
	}
}

// ─── BeginSignup ──────────────────────────────────────────────────────────────

func TestBeginSignup_Disabled_ReturnsErrUnsupportedMethod(t *testing.T) {
	svc := NewAuthService(AuthServiceConfig{
		Users:      &fakeUserStore{},
		TokenCodec: stubTokenCodec{},
		EmailTOTP:  EmailTOTPConfig{Enabled: false},
		Clock:      fixedClock,
	})
	_, err := svc.BeginSignup(context.Background(), BeginSignupInput{Email: "new@example.com"}, "/verify")
	if !errors.Is(err, ErrUnsupportedMethod) {
		t.Errorf("BeginSignup() error = %v, want ErrUnsupportedMethod", err)
	}
}

func TestBeginSignup_NewUser_SendsNotification(t *testing.T) {
	store := &fakeUserStore{getUserByEmailErr: ErrUserNotFound}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	flow, err := svc.BeginSignup(context.Background(), BeginSignupInput{
		Email: "new@example.com", DisplayName: "New User",
	}, "/verify")
	if err != nil {
		t.Fatalf("BeginSignup() error = %v", err)
	}
	if flow.Purpose != FlowSignup {
		t.Errorf("flow.Purpose = %q, want %q", flow.Purpose, FlowSignup)
	}
	if !notifier.called {
		t.Error("notifier.SendVerification was not called for new user")
	}
	if notifier.input.Purpose != FlowSignup {
		t.Errorf("delivery purpose = %q, want %q", notifier.input.Purpose, FlowSignup)
	}
}

func TestBeginSignup_ExistingVerifiedUser_SendsAlreadyRegistered(t *testing.T) {
	store := &fakeUserStore{
		getUserByEmailResult: User{ID: uuid.New(), Email: "taken@example.com", Verified: true},
	}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	flow, err := svc.BeginSignup(context.Background(), BeginSignupInput{
		Email: "taken@example.com", DisplayName: "User",
	}, "/verify")
	if err != nil {
		t.Fatalf("BeginSignup() error = %v", err)
	}
	// Response must look identical to a real signup (anti-enumeration).
	if flow.Purpose != FlowSignup {
		t.Errorf("flow.Purpose = %q, want %q", flow.Purpose, FlowSignup)
	}
	if !notifier.called {
		t.Error("notifier.SendVerification was not called")
	}
	if notifier.input.Purpose != FlowAlreadyRegistered {
		t.Errorf("delivery purpose = %q, want %q", notifier.input.Purpose, FlowAlreadyRegistered)
	}
}

func TestBeginSignup_ExistingUnverifiedUser_ProceedsNormally(t *testing.T) {
	store := &fakeUserStore{
		getUserByEmailResult: User{ID: uuid.New(), Email: "pending@example.com", Verified: false},
	}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	flow, err := svc.BeginSignup(context.Background(), BeginSignupInput{
		Email: "pending@example.com", DisplayName: "Pending",
	}, "/verify")
	if err != nil {
		t.Fatalf("BeginSignup() error = %v", err)
	}
	if flow.Purpose != FlowSignup {
		t.Errorf("flow.Purpose = %q, want %q", flow.Purpose, FlowSignup)
	}
	if !notifier.called {
		t.Error("notifier.SendVerification was not called for unverified user re-signup")
	}
	if notifier.input.Purpose != FlowSignup {
		t.Errorf("delivery purpose = %q, want %q", notifier.input.Purpose, FlowSignup)
	}
}

func TestBeginSignup_DBLookupError_ReturnsError(t *testing.T) {
	dbErr := errors.New("db read failure")
	store := &fakeUserStore{getUserByEmailErr: dbErr}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	_, err := svc.BeginSignup(context.Background(), BeginSignupInput{
		Email: "user@example.com", DisplayName: "User",
	}, "/verify")
	if !errors.Is(err, dbErr) {
		t.Errorf("BeginSignup() error = %v, want wrapped %v", err, dbErr)
	}
}

// ─── BeginEmailChange ─────────────────────────────────────────────────────────

func TestBeginEmailChange_Disabled_ReturnsErrUnsupportedMethod(t *testing.T) {
	svc := NewAuthService(AuthServiceConfig{
		Users:      &fakeUserStore{},
		TokenCodec: stubTokenCodec{},
		EmailTOTP:  EmailTOTPConfig{Enabled: false},
		Clock:      fixedClock,
	})
	_, err := svc.BeginEmailChange(context.Background(), uuid.New(), BeginEmailChangeInput{Email: "new@example.com"}, "/verify")
	if !errors.Is(err, ErrUnsupportedMethod) {
		t.Errorf("BeginEmailChange() error = %v, want ErrUnsupportedMethod", err)
	}
}

func TestBeginEmailChange_UserLookupFails_ReturnsError(t *testing.T) {
	dbErr := errors.New("db read failure")
	store := &fakeUserStore{getUserByIDErr: dbErr}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	_, err := svc.BeginEmailChange(context.Background(), uuid.New(), BeginEmailChangeInput{Email: "new@example.com"}, "/verify")
	if !errors.Is(err, dbErr) {
		t.Errorf("BeginEmailChange() error = %v, want wrapped %v", err, dbErr)
	}
}

func TestBeginEmailChange_SameEmail_ReturnsErrEmailTaken(t *testing.T) {
	store := &fakeUserStore{
		getUserByIDResult: User{ID: uuid.New(), Email: "same@example.com"},
	}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	_, err := svc.BeginEmailChange(context.Background(), uuid.New(), BeginEmailChangeInput{Email: "same@example.com"}, "/verify")
	if !errors.Is(err, ErrEmailTaken) {
		t.Errorf("BeginEmailChange() error = %v, want ErrEmailTaken", err)
	}
}

func TestBeginEmailChange_SameEmail_CaseInsensitive_ReturnsErrEmailTaken(t *testing.T) {
	store := &fakeUserStore{
		getUserByIDResult: User{ID: uuid.New(), Email: "user@example.com"},
	}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	_, err := svc.BeginEmailChange(context.Background(), uuid.New(), BeginEmailChangeInput{Email: "USER@EXAMPLE.COM"}, "/verify")
	if !errors.Is(err, ErrEmailTaken) {
		t.Errorf("BeginEmailChange() error = %v, want ErrEmailTaken", err)
	}
}

func TestBeginEmailChange_TargetEmailAlreadyTaken_ReturnsErrEmailTaken(t *testing.T) {
	store := &fakeUserStore{
		getUserByIDResult:    User{ID: uuid.New(), Email: "current@example.com"},
		getUserByEmailResult: User{ID: uuid.New(), Email: "taken@example.com"},
	}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	_, err := svc.BeginEmailChange(context.Background(), uuid.New(), BeginEmailChangeInput{Email: "taken@example.com"}, "/verify")
	if !errors.Is(err, ErrEmailTaken) {
		t.Errorf("BeginEmailChange() error = %v, want ErrEmailTaken", err)
	}
}

func TestBeginEmailChange_EmailLookupDBError_ReturnsError(t *testing.T) {
	dbErr := errors.New("db read failure")
	store := &fakeUserStore{
		getUserByIDResult: User{ID: uuid.New(), Email: "current@example.com"},
		getUserByEmailErr: dbErr,
	}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	_, err := svc.BeginEmailChange(context.Background(), uuid.New(), BeginEmailChangeInput{Email: "new@example.com"}, "/verify")
	if !errors.Is(err, dbErr) {
		t.Errorf("BeginEmailChange() error = %v, want wrapped %v", err, dbErr)
	}
}

func TestBeginEmailChange_Success_SendsNotification(t *testing.T) {
	store := &fakeUserStore{
		getUserByIDResult: User{ID: uuid.New(), Email: "old@example.com"},
		getUserByEmailErr: ErrUserNotFound,
	}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	flow, err := svc.BeginEmailChange(context.Background(), uuid.New(), BeginEmailChangeInput{Email: "new@example.com"}, "/verify")
	if err != nil {
		t.Fatalf("BeginEmailChange() error = %v", err)
	}
	if flow.Purpose != FlowEmailChange {
		t.Errorf("flow.Purpose = %q, want %q", flow.Purpose, FlowEmailChange)
	}
	if !notifier.called {
		t.Error("notifier.SendVerification was not called")
	}
	if notifier.input.Purpose != FlowEmailChange {
		t.Errorf("delivery purpose = %q, want %q", notifier.input.Purpose, FlowEmailChange)
	}
}

// ─── ResendPendingFlow ────────────────────────────────────────────────────────

func TestResendPendingFlow_UnknownPurpose_ReturnsErrUnsupportedMethod(t *testing.T) {
	svc := newTestAuthService(&fakeUserStore{}, &fakeNotifier{})
	_, err := svc.ResendPendingFlow(context.Background(), PendingFlow{Purpose: "unknown"}, "/verify")
	if !errors.Is(err, ErrUnsupportedMethod) {
		t.Errorf("ResendPendingFlow() error = %v, want ErrUnsupportedMethod", err)
	}
}

func TestResendPendingFlow_Signup_DelegatesToBeginSignup(t *testing.T) {
	store := &fakeUserStore{getUserByEmailErr: ErrUserNotFound}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	flow, err := svc.ResendPendingFlow(context.Background(), PendingFlow{
		Purpose: FlowSignup, Email: "new@example.com", DisplayName: "User",
	}, "/verify")
	if err != nil {
		t.Fatalf("ResendPendingFlow(FlowSignup) error = %v", err)
	}
	if flow.Purpose != FlowSignup {
		t.Errorf("flow.Purpose = %q, want %q", flow.Purpose, FlowSignup)
	}
	if !notifier.called {
		t.Error("notifier.SendVerification was not called")
	}
}

func TestResendPendingFlow_Signin_DelegatesToBeginSignin(t *testing.T) {
	store := &fakeUserStore{
		getUserByEmailResult: User{ID: uuid.New(), Email: "user@example.com", Verified: true},
	}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	flow, err := svc.ResendPendingFlow(context.Background(), PendingFlow{
		Purpose: FlowSignin, Email: "user@example.com",
	}, "/verify")
	if err != nil {
		t.Fatalf("ResendPendingFlow(FlowSignin) error = %v", err)
	}
	if flow.Purpose != FlowSignin {
		t.Errorf("flow.Purpose = %q, want %q", flow.Purpose, FlowSignin)
	}
}

func TestResendPendingFlow_EmailChange_DelegatesToBeginEmailChange(t *testing.T) {
	userID := uuid.New()
	store := &fakeUserStore{
		getUserByIDResult: User{ID: userID, Email: "old@example.com"},
		getUserByEmailErr: ErrUserNotFound,
	}
	notifier := &fakeNotifier{}
	svc := newTestAuthService(store, notifier)

	flow, err := svc.ResendPendingFlow(context.Background(), PendingFlow{
		Purpose: FlowEmailChange, Email: "new@example.com", UserID: userID,
	}, "/verify")
	if err != nil {
		t.Fatalf("ResendPendingFlow(FlowEmailChange) error = %v", err)
	}
	if flow.Purpose != FlowEmailChange {
		t.Errorf("flow.Purpose = %q, want %q", flow.Purpose, FlowEmailChange)
	}
	if !notifier.called {
		t.Error("notifier.SendVerification was not called")
	}
}

// ─── VerifyPendingFlow ────────────────────────────────────────────────────────

func TestVerifyPendingFlow_MaxAttempts_ReturnsErrTooManyAttempts(t *testing.T) {
	svc := newTestAuthService(&fakeUserStore{}, &fakeNotifier{})
	flow := PendingFlow{
		Purpose:   FlowSignin,
		Attempts:  maxVerifyAttempts,
		ExpiresAt: fixedClock().Add(5 * time.Minute),
	}
	_, _, err := svc.VerifyPendingFlow(context.Background(), flow, VerifyInput{Code: "123456"})
	if !errors.Is(err, ErrTooManyAttempts) {
		t.Errorf("VerifyPendingFlow() error = %v, want ErrTooManyAttempts", err)
	}
}

func TestVerifyPendingFlow_InvalidCode_IncrementsAttempts(t *testing.T) {
	svc := newTestAuthService(&fakeUserStore{}, &fakeNotifier{})
	flow, _ := validFlow(t, FlowSignin, "user@example.com", uuid.Nil)

	_, updated, err := svc.VerifyPendingFlow(context.Background(), flow, VerifyInput{Code: "000000"})
	if !errors.Is(err, ErrInvalidVerificationCode) {
		t.Errorf("VerifyPendingFlow() error = %v, want ErrInvalidVerificationCode", err)
	}
	if updated.Attempts != 1 {
		t.Errorf("updated.Attempts = %d, want 1", updated.Attempts)
	}
}

func TestVerifyPendingFlow_Expired_ReturnsErrVerificationExpired(t *testing.T) {
	svc := newTestAuthService(&fakeUserStore{}, &fakeNotifier{})
	now := fixedClock()
	flow := PendingFlow{
		Purpose:   FlowSignin,
		ExpiresAt: now.Add(-1 * time.Minute),
		IssuedAt:  now.Add(-6 * time.Minute),
		Secret:    "DUMMY",
	}
	_, _, err := svc.VerifyPendingFlow(context.Background(), flow, VerifyInput{Code: "123456"})
	if !errors.Is(err, ErrVerificationExpired) {
		t.Errorf("VerifyPendingFlow() error = %v, want ErrVerificationExpired", err)
	}
}

func TestVerifyPendingFlow_Signin_Success(t *testing.T) {
	userID := uuid.New()
	store := &fakeUserStore{
		getUserByEmailResult: User{ID: userID, Email: "user@example.com", Verified: true},
	}
	svc := newTestAuthService(store, &fakeNotifier{})
	flow, code := validFlow(t, FlowSignin, "user@example.com", uuid.Nil)

	result, updated, err := svc.VerifyPendingFlow(context.Background(), flow, VerifyInput{Code: code})
	if err != nil {
		t.Fatalf("VerifyPendingFlow() error = %v", err)
	}
	if result.Purpose != FlowSignin {
		t.Errorf("result.Purpose = %q, want %q", result.Purpose, FlowSignin)
	}
	if result.User.ID != userID {
		t.Errorf("result.User.ID = %v, want %v", result.User.ID, userID)
	}
	if result.Method != string(MethodEmailTOTP) {
		t.Errorf("result.Method = %q, want %q", result.Method, MethodEmailTOTP)
	}
	if updated.Attempts != 1 {
		t.Errorf("updated.Attempts = %d, want 1", updated.Attempts)
	}
}

func TestVerifyPendingFlow_Signin_UserNotFound_ReturnsErrInvalidCredentials(t *testing.T) {
	store := &fakeUserStore{getUserByEmailErr: ErrUserNotFound}
	svc := newTestAuthService(store, &fakeNotifier{})
	flow, code := validFlow(t, FlowSignin, "ghost@example.com", uuid.Nil)

	_, _, err := svc.VerifyPendingFlow(context.Background(), flow, VerifyInput{Code: code})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("VerifyPendingFlow() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestVerifyPendingFlow_Signin_UnverifiedUser_ReturnsErrInvalidCredentials(t *testing.T) {
	store := &fakeUserStore{
		getUserByEmailResult: User{ID: uuid.New(), Email: "unverified@example.com", Verified: false},
	}
	svc := newTestAuthService(store, &fakeNotifier{})
	flow, code := validFlow(t, FlowSignin, "unverified@example.com", uuid.Nil)

	_, _, err := svc.VerifyPendingFlow(context.Background(), flow, VerifyInput{Code: code})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("VerifyPendingFlow() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestVerifyPendingFlow_Signup_Success(t *testing.T) {
	userID := uuid.New()
	store := &fakeUserStore{
		createUserResult: User{ID: userID, Email: "new@example.com", Verified: true},
	}
	svc := newTestAuthService(store, &fakeNotifier{})
	flow, code := validFlow(t, FlowSignup, "new@example.com", uuid.Nil)
	flow.DisplayName = "New User"

	result, _, err := svc.VerifyPendingFlow(context.Background(), flow, VerifyInput{Code: code})
	if err != nil {
		t.Fatalf("VerifyPendingFlow() error = %v", err)
	}
	if result.Purpose != FlowSignup {
		t.Errorf("result.Purpose = %q, want %q", result.Purpose, FlowSignup)
	}
	if result.User.ID != userID {
		t.Errorf("result.User.ID = %v, want %v", result.User.ID, userID)
	}
}

func TestVerifyPendingFlow_Signup_EmailTaken_VerifiedExisting_Succeeds(t *testing.T) {
	// Race: CreateUser returns ErrEmailTaken but the existing user is verified —
	// idempotent: treat as successful signup.
	existingID := uuid.New()
	store := &fakeUserStore{
		createUserErr:        ErrEmailTaken,
		getUserByEmailResult: User{ID: existingID, Email: "race@example.com", Verified: true},
	}
	svc := newTestAuthService(store, &fakeNotifier{})
	flow, code := validFlow(t, FlowSignup, "race@example.com", uuid.Nil)

	result, _, err := svc.VerifyPendingFlow(context.Background(), flow, VerifyInput{Code: code})
	if err != nil {
		t.Fatalf("VerifyPendingFlow() error = %v, want nil (idempotent)", err)
	}
	if result.User.ID != existingID {
		t.Errorf("result.User.ID = %v, want %v", result.User.ID, existingID)
	}
}

func TestVerifyPendingFlow_EmailChange_Success(t *testing.T) {
	userID := uuid.New()
	updatedUser := User{ID: userID, Email: "new@example.com", Verified: true}
	store := &fakeUserStore{
		getUserByIDResult:     User{ID: userID, Email: "old@example.com"},
		updateUserEmailResult: updatedUser,
	}
	svc := newTestAuthService(store, &fakeNotifier{})
	flow, code := validFlow(t, FlowEmailChange, "new@example.com", userID)

	result, _, err := svc.VerifyPendingFlow(context.Background(), flow, VerifyInput{Code: code})
	if err != nil {
		t.Fatalf("VerifyPendingFlow() error = %v", err)
	}
	if result.Purpose != FlowEmailChange {
		t.Errorf("result.Purpose = %q, want %q", result.Purpose, FlowEmailChange)
	}
	if result.User.Email != "new@example.com" {
		t.Errorf("result.User.Email = %q, want %q", result.User.Email, "new@example.com")
	}
}

// ─── VerifyToken ──────────────────────────────────────────────────────────────

func TestVerifyToken_Replay_ReturnsErrTokenConsumed(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	secret, err := randomSecret()
	if err != nil {
		t.Fatalf("randomSecret() error = %v", err)
	}
	code, err := generateTOTPCode(secret, fixedTime, 300)
	if err != nil {
		t.Fatalf("generateTOTPCode() error = %v", err)
	}

	token := VerificationToken{
		Purpose:   FlowSignin,
		Email:     "user@example.com",
		Secret:    secret,
		IssuedAt:  fixedTime,
		ExpiresAt: fixedTime.Add(5 * time.Minute),
	}
	store := &fakeUserStore{
		getUserByEmailResult: User{ID: uuid.New(), Email: "user@example.com", Verified: true},
	}
	svc := NewAuthService(AuthServiceConfig{
		Users:      store,
		TokenCodec: fixedTokenCodec{token: token},
		NonceStore: newFakeNonceStore(),
		EmailTOTP:  EmailTOTPConfig{Enabled: true, PeriodSeconds: 300},
		Clock:      func() time.Time { return fixedTime },
	})

	_, err = svc.VerifyToken(context.Background(), "fixed-token", VerifyInput{Code: code})
	if err != nil {
		t.Fatalf("first VerifyToken() error = %v, want nil", err)
	}

	_, err = svc.VerifyToken(context.Background(), "fixed-token", VerifyInput{Code: code})
	if !errors.Is(err, ErrTokenConsumed) {
		t.Errorf("second VerifyToken() error = %v, want ErrTokenConsumed", err)
	}
}

func TestVerifyToken_FirstUse_Succeeds(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	secret, err := randomSecret()
	if err != nil {
		t.Fatalf("randomSecret() error = %v", err)
	}
	code, err := generateTOTPCode(secret, fixedTime, 300)
	if err != nil {
		t.Fatalf("generateTOTPCode() error = %v", err)
	}

	token := VerificationToken{
		Purpose:   FlowSignin,
		Email:     "user@example.com",
		Secret:    secret,
		IssuedAt:  fixedTime,
		ExpiresAt: fixedTime.Add(5 * time.Minute),
	}
	store := &fakeUserStore{
		getUserByEmailResult: User{ID: uuid.New(), Email: "user@example.com", Verified: true},
	}
	svc := NewAuthService(AuthServiceConfig{
		Users:      store,
		TokenCodec: fixedTokenCodec{token: token},
		NonceStore: newFakeNonceStore(),
		EmailTOTP:  EmailTOTPConfig{Enabled: true, PeriodSeconds: 300},
		Clock:      func() time.Time { return fixedTime },
	})

	result, err := svc.VerifyToken(context.Background(), "fixed-token", VerifyInput{Code: code})
	if err != nil {
		t.Fatalf("VerifyToken() error = %v, want nil", err)
	}
	if result.Purpose != FlowSignin {
		t.Errorf("result.Purpose = %q, want %q", result.Purpose, FlowSignin)
	}
}

func TestVerifyToken_InvalidDecode_ReturnsError(t *testing.T) {
	svc := NewAuthService(AuthServiceConfig{
		Users:      &fakeUserStore{},
		TokenCodec: failingTokenCodec{err: ErrVerificationMissing},
		EmailTOTP:  EmailTOTPConfig{Enabled: true, PeriodSeconds: 300},
		Clock:      fixedClock,
	})
	_, err := svc.VerifyToken(context.Background(), "bad-token", VerifyInput{Code: "123456"})
	if !errors.Is(err, ErrVerificationMissing) {
		t.Errorf("VerifyToken() error = %v, want ErrVerificationMissing", err)
	}
}

func TestVerifyToken_NonceStoreHasConsumedError_ReturnsError(t *testing.T) {
	now := fixedClock()
	secret, _ := randomSecret()
	code, _ := generateTOTPCode(secret, now, 300)
	token := VerificationToken{
		Purpose:   FlowSignin,
		Email:     "user@example.com",
		Secret:    secret,
		IssuedAt:  now,
		ExpiresAt: now.Add(5 * time.Minute),
	}
	nonceErr := errors.New("nonce store unavailable")
	svc := NewAuthService(AuthServiceConfig{
		Users:      &fakeUserStore{},
		TokenCodec: fixedTokenCodec{token: token},
		NonceStore: &fakeNonceStore{consumed: make(map[string]struct{}), hasConsumedErr: nonceErr},
		EmailTOTP:  EmailTOTPConfig{Enabled: true, PeriodSeconds: 300},
		Clock:      fixedClock,
	})
	_, err := svc.VerifyToken(context.Background(), "fixed-token", VerifyInput{Code: code})
	if !errors.Is(err, nonceErr) {
		t.Errorf("VerifyToken() error = %v, want wrapped %v", err, nonceErr)
	}
}

func TestVerifyToken_NonceStoreMarkConsumedError_ReturnsError(t *testing.T) {
	now := fixedClock()
	secret, _ := randomSecret()
	code, _ := generateTOTPCode(secret, now, 300)
	token := VerificationToken{
		Purpose:   FlowSignin,
		Email:     "user@example.com",
		Secret:    secret,
		IssuedAt:  now,
		ExpiresAt: now.Add(5 * time.Minute),
	}
	markErr := errors.New("nonce store write failure")
	store := &fakeUserStore{
		getUserByEmailResult: User{ID: uuid.New(), Email: "user@example.com", Verified: true},
	}
	svc := NewAuthService(AuthServiceConfig{
		Users:      store,
		TokenCodec: fixedTokenCodec{token: token},
		NonceStore: &fakeNonceStore{consumed: make(map[string]struct{}), markConsumedErr: markErr},
		EmailTOTP:  EmailTOTPConfig{Enabled: true, PeriodSeconds: 300},
		Clock:      fixedClock,
	})
	_, err := svc.VerifyToken(context.Background(), "fixed-token", VerifyInput{Code: code})
	if !errors.Is(err, markErr) {
		t.Errorf("VerifyToken() error = %v, want wrapped %v", err, markErr)
	}
}

func TestVerifyToken_NilNonceStore_Success(t *testing.T) {
	now := fixedClock()
	secret, _ := randomSecret()
	code, _ := generateTOTPCode(secret, now, 300)
	token := VerificationToken{
		Purpose:   FlowSignin,
		Email:     "user@example.com",
		Secret:    secret,
		IssuedAt:  now,
		ExpiresAt: now.Add(5 * time.Minute),
	}
	store := &fakeUserStore{
		getUserByEmailResult: User{ID: uuid.New(), Email: "user@example.com", Verified: true},
	}
	svc := NewAuthService(AuthServiceConfig{
		Users:      store,
		TokenCodec: fixedTokenCodec{token: token},
		NonceStore: nil,
		EmailTOTP:  EmailTOTPConfig{Enabled: true, PeriodSeconds: 300},
		Clock:      fixedClock,
	})
	result, err := svc.VerifyToken(context.Background(), "any-token", VerifyInput{Code: code})
	if err != nil {
		t.Fatalf("VerifyToken() error = %v, want nil", err)
	}
	if result.Purpose != FlowSignin {
		t.Errorf("result.Purpose = %q, want %q", result.Purpose, FlowSignin)
	}
}

// ─── VerifyPageState ──────────────────────────────────────────────────────────

func TestVerifyPageState_InvalidToken_ReturnsError(t *testing.T) {
	svc := NewAuthService(AuthServiceConfig{
		Users:      &fakeUserStore{},
		TokenCodec: failingTokenCodec{err: ErrVerificationMissing},
		EmailTOTP:  EmailTOTPConfig{Enabled: true, PeriodSeconds: 300},
		Clock:      fixedClock,
	})
	_, err := svc.VerifyPageState("bad-token")
	if !errors.Is(err, ErrVerificationMissing) {
		t.Errorf("VerifyPageState() error = %v, want ErrVerificationMissing", err)
	}
}

func TestVerifyPageState_ValidToken_ReturnsState(t *testing.T) {
	token := VerificationToken{Purpose: FlowSignin, Email: "user@example.com"}
	svc := NewAuthService(AuthServiceConfig{
		Users:      &fakeUserStore{},
		TokenCodec: fixedTokenCodec{token: token},
		EmailTOTP:  EmailTOTPConfig{Enabled: true, PeriodSeconds: 300},
		Clock:      fixedClock,
	})
	state, err := svc.VerifyPageState("fixed-token")
	if err != nil {
		t.Fatalf("VerifyPageState() error = %v", err)
	}
	if state.Purpose != FlowSignin {
		t.Errorf("state.Purpose = %q, want %q", state.Purpose, FlowSignin)
	}
	if state.Email != "user@example.com" {
		t.Errorf("state.Email = %q, want %q", state.Email, "user@example.com")
	}
	if state.Token != "fixed-token" {
		t.Errorf("state.Token = %q, want %q", state.Token, "fixed-token")
	}
}

// ─── deliver (notifier error) ─────────────────────────────────────────────────

func TestBeginSignup_NotifierError_ReturnsError(t *testing.T) {
	notifierErr := errors.New("smtp unavailable")
	store := &fakeUserStore{getUserByEmailErr: ErrUserNotFound}
	notifier := &fakeNotifier{err: notifierErr}
	svc := newTestAuthService(store, notifier)

	_, err := svc.BeginSignup(context.Background(), BeginSignupInput{
		Email: "new@example.com", DisplayName: "User",
	}, "/verify")
	if !errors.Is(err, notifierErr) {
		t.Errorf("BeginSignup() error = %v, want wrapped %v", err, notifierErr)
	}
}

// ─── finishSignin DB error ────────────────────────────────────────────────────

func TestVerifyPendingFlow_Signin_DBError_ReturnsError(t *testing.T) {
	dbErr := errors.New("db read failure")
	store := &fakeUserStore{getUserByEmailErr: dbErr}
	svc := newTestAuthService(store, &fakeNotifier{})
	flow, code := validFlow(t, FlowSignin, "user@example.com", uuid.Nil)

	_, _, err := svc.VerifyPendingFlow(context.Background(), flow, VerifyInput{Code: code})
	if !errors.Is(err, dbErr) {
		t.Errorf("VerifyPendingFlow() error = %v, want wrapped %v", err, dbErr)
	}
}

// ─── finishSignup DB error ────────────────────────────────────────────────────

func TestVerifyPendingFlow_Signup_CreateUserDBError_ReturnsError(t *testing.T) {
	dbErr := errors.New("constraint violation")
	store := &fakeUserStore{createUserErr: dbErr}
	svc := newTestAuthService(store, &fakeNotifier{})
	flow, code := validFlow(t, FlowSignup, "new@example.com", uuid.Nil)

	_, _, err := svc.VerifyPendingFlow(context.Background(), flow, VerifyInput{Code: code})
	if !errors.Is(err, dbErr) {
		t.Errorf("VerifyPendingFlow() error = %v, want wrapped %v", err, dbErr)
	}
}

// ─── finishEmailChange error paths ───────────────────────────────────────────

func TestVerifyPendingFlow_EmailChange_UserLookupError_ReturnsError(t *testing.T) {
	dbErr := errors.New("db read failure")
	store := &fakeUserStore{getUserByIDErr: dbErr}
	svc := newTestAuthService(store, &fakeNotifier{})
	flow, code := validFlow(t, FlowEmailChange, "new@example.com", uuid.New())

	_, _, err := svc.VerifyPendingFlow(context.Background(), flow, VerifyInput{Code: code})
	if !errors.Is(err, dbErr) {
		t.Errorf("VerifyPendingFlow() error = %v, want wrapped %v", err, dbErr)
	}
}

func TestVerifyPendingFlow_EmailChange_AlreadySameEmail_ReturnsUser(t *testing.T) {
	// Email was already updated before verification completed (idempotent).
	userID := uuid.New()
	store := &fakeUserStore{
		getUserByIDResult: User{ID: userID, Email: "new@example.com"},
	}
	svc := newTestAuthService(store, &fakeNotifier{})
	flow, code := validFlow(t, FlowEmailChange, "new@example.com", userID)

	result, _, err := svc.VerifyPendingFlow(context.Background(), flow, VerifyInput{Code: code})
	if err != nil {
		t.Fatalf("VerifyPendingFlow() error = %v, want nil", err)
	}
	if result.Purpose != FlowEmailChange {
		t.Errorf("result.Purpose = %q, want %q", result.Purpose, FlowEmailChange)
	}
	if result.User.ID != userID {
		t.Errorf("result.User.ID = %v, want %v", result.User.ID, userID)
	}
}

func TestVerifyPendingFlow_EmailChange_UpdateEmailError_ReturnsError(t *testing.T) {
	dbErr := errors.New("constraint violation")
	userID := uuid.New()
	store := &fakeUserStore{
		getUserByIDResult:  User{ID: userID, Email: "old@example.com"},
		updateUserEmailErr: dbErr,
	}
	svc := newTestAuthService(store, &fakeNotifier{})
	flow, code := validFlow(t, FlowEmailChange, "new@example.com", userID)

	_, _, err := svc.VerifyPendingFlow(context.Background(), flow, VerifyInput{Code: code})
	if !errors.Is(err, dbErr) {
		t.Errorf("VerifyPendingFlow() error = %v, want wrapped %v", err, dbErr)
	}
}

func TestVerifyPendingFlow_EmailChange_ErrEmailTaken_RaceResolved(t *testing.T) {
	// UpdateUserEmail races with another update: ErrEmailTaken is returned, but
	// a follow-up GetUserByID shows the email is already the target — success.
	userID := uuid.New()
	callCount := 0
	store := &fakeUserStore{
		updateUserEmailErr: ErrEmailTaken,
		getUserByIDFn: func(_ uuid.UUID) (User, error) {
			callCount++
			if callCount == 1 {
				return User{ID: userID, Email: "old@example.com"}, nil
			}
			return User{ID: userID, Email: "new@example.com"}, nil
		},
	}
	svc := newTestAuthService(store, &fakeNotifier{})
	flow, code := validFlow(t, FlowEmailChange, "new@example.com", userID)

	result, _, err := svc.VerifyPendingFlow(context.Background(), flow, VerifyInput{Code: code})
	if err != nil {
		t.Fatalf("VerifyPendingFlow() error = %v, want nil (race resolved)", err)
	}
	if result.User.Email != "new@example.com" {
		t.Errorf("result.User.Email = %q, want %q", result.User.Email, "new@example.com")
	}
}
