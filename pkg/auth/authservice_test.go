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

// fixedTokenCodec always decodes to the same pre-set VerificationToken.
type fixedTokenCodec struct {
	token VerificationToken
}

func (f fixedTokenCodec) Encode(VerificationToken) (string, error) { return "fixed-token", nil }
func (f fixedTokenCodec) Decode(string) (VerificationToken, error)  { return f.token, nil }

// fakeUserStore implements UserWriter for testing. Only GetUserByEmail is
// functional; all write methods return zero values and nil error.
type fakeUserStore struct {
	getUserByEmailResult User
	getUserByEmailErr    error
}

func (f *fakeUserStore) GetUserByEmail(_ context.Context, _ string) (User, error) {
	return f.getUserByEmailResult, f.getUserByEmailErr
}
func (f *fakeUserStore) GetUserByID(_ context.Context, _ uuid.UUID) (User, error) {
	return User{}, nil
}
func (f *fakeUserStore) CreateUser(_ context.Context, _, _ string, _ Role, _ bool) (User, error) {
	return User{}, nil
}
func (f *fakeUserStore) UpdateUserEmail(_ context.Context, _ uuid.UUID, _ string) (User, error) {
	return User{}, nil
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

// fakeNotifier records whether SendVerification was called.
type fakeNotifier struct {
	called bool
}

func (f *fakeNotifier) SendVerification(_ context.Context, _ DeliveryInput) error {
	f.called = true
	return nil
}

// fakeNonceStore is an in-memory TokenNonceStore for testing.
type fakeNonceStore struct {
	consumed map[string]struct{}
}

func newFakeNonceStore() *fakeNonceStore {
	return &fakeNonceStore{consumed: make(map[string]struct{})}
}

func (f *fakeNonceStore) HasConsumed(_ context.Context, key string) (bool, error) {
	_, ok := f.consumed[key]
	return ok, nil
}

func (f *fakeNonceStore) MarkConsumed(_ context.Context, key string, _ time.Duration) error {
	f.consumed[key] = struct{}{}
	return nil
}

func newTestAuthService(store *fakeUserStore, notifier *fakeNotifier) *AuthService {
	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return NewAuthService(AuthServiceConfig{
		Users:      store,
		Notifier:   notifier,
		TokenCodec: stubTokenCodec{},
		NonceStore: newFakeNonceStore(),
		EmailTOTP: EmailTOTPConfig{
			Enabled:       true,
			PeriodSeconds: 300,
		},
		Clock: func() time.Time { return fixedTime },
	})
}

func TestBeginSignin_VerifiedUser_TriggersDelivery(t *testing.T) {
	store := &fakeUserStore{
		getUserByEmailResult: User{
			ID:       uuid.New(),
			Email:    "user@example.com",
			Verified: true,
		},
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
	store := &fakeUserStore{
		getUserByEmailErr: ErrUserNotFound,
	}
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
		getUserByEmailResult: User{
			ID:       uuid.New(),
			Email:    "unverified@example.com",
			Verified: false,
		},
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
			name: "unknown user (ErrUserNotFound)",
			store: &fakeUserStore{
				getUserByEmailErr: ErrUserNotFound,
			},
		},
		{
			name: "unverified user",
			store: &fakeUserStore{
				getUserByEmailResult: User{
					ID:       uuid.New(),
					Email:    testEmail,
					Verified: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture slog output via a buffer. No t.Parallel() — global state.
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

			logOutput := buf.String()
			if strings.Contains(logOutput, testEmail) {
				t.Errorf("log output contains submitted email %q (PII leak): %s", testEmail, logOutput)
			}
		})
	}
}

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
		getUserByEmailResult: User{
			ID:       uuid.New(),
			Email:    "user@example.com",
			Verified: true,
		},
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
		getUserByEmailResult: User{
			ID:       uuid.New(),
			Email:    "user@example.com",
			Verified: true,
		},
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
