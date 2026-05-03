package authweb

import (
	"context"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/validate"
	"github.com/google/uuid"
)

type stubUserStore struct{}

func (stubUserStore) GetUserByID(context.Context, uuid.UUID) (auth.User, error) {
	return auth.User{}, nil
}
func (stubUserStore) GetUserByEmail(context.Context, string) (auth.User, error) {
	return auth.User{}, nil
}
func (stubUserStore) CreateUser(context.Context, string, string, auth.Role, bool) (auth.User, error) {
	return auth.User{}, nil
}
func (stubUserStore) UpdateUserEmail(context.Context, uuid.UUID, string) (auth.User, error) {
	return auth.User{}, nil
}
func (stubUserStore) SetWebAuthnID(context.Context, uuid.UUID, []byte) (auth.User, error) {
	return auth.User{}, nil
}
func (stubUserStore) SetWebAuthnIDIfNull(context.Context, uuid.UUID, []byte) (auth.User, error) {
	return auth.User{}, nil
}
func (stubUserStore) GetUserByWebAuthnID(context.Context, []byte) (auth.User, error) {
	return auth.User{}, nil
}

type stubAdminStore struct{}

func (stubAdminStore) GetUserByID(context.Context, uuid.UUID) (auth.User, error) {
	return auth.User{}, nil
}
func (stubAdminStore) GetUserByEmail(context.Context, string) (auth.User, error) {
	return auth.User{}, nil
}
func (stubAdminStore) ListUsers(context.Context, int32, int32) ([]auth.User, error) {
	return nil, nil
}
func (stubAdminStore) UpdateUser(context.Context, uuid.UUID, string, string, string) (auth.User, error) {
	return auth.User{}, nil
}
func (stubAdminStore) DeleteUser(context.Context, uuid.UUID) error          { return nil }
func (stubAdminStore) CountUsers(context.Context) (int64, error)            { return 0, nil }
func (stubAdminStore) HasAdmin(context.Context) (bool, error)               { return false, nil }
func (stubAdminStore) IsLastAdmin(context.Context, uuid.UUID) (bool, error) { return false, nil }
func (stubAdminStore) ElevateToAdmin(context.Context, uuid.UUID) (auth.User, error) {
	return auth.User{}, nil
}

type stubCredentialStore struct{}

func (stubCredentialStore) CreateCredential(context.Context, auth.PasskeyCredential) (auth.PasskeyCredential, error) {
	return auth.PasskeyCredential{}, nil
}
func (stubCredentialStore) GetByCredentialID(context.Context, []byte) (auth.PasskeyCredential, error) {
	return auth.PasskeyCredential{}, nil
}
func (stubCredentialStore) GetByIDForUser(context.Context, uuid.UUID, uuid.UUID) (auth.PasskeyCredential, error) {
	return auth.PasskeyCredential{}, nil
}
func (stubCredentialStore) ListByUserID(context.Context, uuid.UUID) ([]auth.PasskeyCredential, error) {
	return nil, nil
}
func (stubCredentialStore) TouchPasskeyCredential(context.Context, uuid.UUID, int64, bool, time.Time) error {
	return nil
}
func (stubCredentialStore) RenameCredential(context.Context, uuid.UUID, uuid.UUID, string) (auth.PasskeyCredential, error) {
	return auth.PasskeyCredential{}, nil
}
func (stubCredentialStore) DeleteCredential(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

type stubRenderer struct{}

func (stubRenderer) SigninPage(*web.Context, SigninPageData) (web.Response, error) {
	return web.Respond(200), nil
}
func (stubRenderer) SignupPage(*web.Context, SignupPageData) (web.Response, error) {
	return web.Respond(200), nil
}
func (stubRenderer) VerifyPage(*web.Context, VerifyPageData) (web.Response, error) {
	return web.Respond(200), nil
}
func (stubRenderer) EmailChangePage(*web.Context, EmailChangePageData) (web.Response, error) {
	return web.Respond(200), nil
}

type stubAdminRenderer struct{}

func (stubAdminRenderer) UsersListPage(*web.Context, UsersListPageData) (web.Response, error) {
	return web.Respond(200), nil
}
func (stubAdminRenderer) UserEditPage(*web.Context, UserEditPageData) (web.Response, error) {
	return web.Respond(200), nil
}
func (stubAdminRenderer) UserRowFragment(*web.Context, auth.User) (web.Response, error) {
	return web.Respond(200), nil
}
func (stubAdminRenderer) BootstrapPage(*web.Context, BootstrapPageData) (web.Response, error) {
	return web.Respond(200), nil
}

type stubTokenCodec struct{}

func (stubTokenCodec) Encode(auth.VerificationToken) (string, error) { return "token", nil }
func (stubTokenCodec) Decode(string) (auth.VerificationToken, error) {
	return auth.VerificationToken{}, nil
}

func validModuleConfig() ModuleConfig {
	return ModuleConfig{
		Router:        router.New(),
		Validator:     validate.New(),
		Users:         stubUserStore{},
		AdminUsers:    stubAdminStore{},
		Renderer:      stubRenderer{},
		AdminRenderer: stubAdminRenderer{},
	}
}

func TestNewModule_ValidatesRequiredDependencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*ModuleConfig)
		wantErr string
	}{
		{
			name: "missing router",
			mutate: func(cfg *ModuleConfig) {
				cfg.Router = nil
			},
			wantErr: "auth: Router is required",
		},
		{
			name: "missing validator",
			mutate: func(cfg *ModuleConfig) {
				cfg.Validator = nil
			},
			wantErr: "auth: Validator is required",
		},
		{
			name: "missing users",
			mutate: func(cfg *ModuleConfig) {
				cfg.Users = nil
			},
			wantErr: "auth: Users (UserWriter) is required",
		},
		{
			name: "missing renderer",
			mutate: func(cfg *ModuleConfig) {
				cfg.Renderer = nil
			},
			wantErr: "auth: Renderer is required",
		},
		{
			name: "missing admin users",
			mutate: func(cfg *ModuleConfig) {
				cfg.AdminUsers = nil
			},
			wantErr: "auth: AdminUsers (AdminStore) is required",
		},
		{
			name: "missing admin renderer",
			mutate: func(cfg *ModuleConfig) {
				cfg.AdminRenderer = nil
			},
			wantErr: "auth: AdminRenderer is required",
		},
		{
			name: "email totp enabled without token codec",
			mutate: func(cfg *ModuleConfig) {
				cfg.Config.EmailTOTP.Enabled = true
			},
			wantErr: "auth: TokenCodec is required when email TOTP is enabled",
		},
		{
			name: "email totp enabled without token nonce store",
			mutate: func(cfg *ModuleConfig) {
				cfg.Config.EmailTOTP.Enabled = true
				cfg.TokenCodec = stubTokenCodec{}
			},
			wantErr: "auth: TokenNonceStore is required when email TOTP is enabled",
		},
		{
			name: "passkeys enabled without credentials",
			mutate: func(cfg *ModuleConfig) {
				cfg.Config.Passkey.Enabled = true
			},
			wantErr: "auth: Credentials (CredentialStore) is required when passkeys are enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validModuleConfig()
			tt.mutate(&cfg)

			_, err := NewModule(cfg)
			if err == nil {
				t.Fatal("NewModule error = nil, want error")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("NewModule error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestNewModule_AllowsOptionalDependenciesWhenFeaturesDisabled(t *testing.T) {
	t.Parallel()

	cfg := validModuleConfig()

	m, err := NewModule(cfg)
	if err != nil {
		t.Fatalf("NewModule error: %v", err)
	}
	if m == nil {
		t.Fatal("NewModule returned nil module")
	}
	if m.AuthHandler == nil {
		t.Fatal("AuthHandler is nil")
	}
	if m.AdminHandler == nil {
		t.Fatal("AdminHandler is nil")
	}
	if m.PasskeyHandler != nil {
		t.Fatal("PasskeyHandler is non-nil with passkeys disabled")
	}
}

func TestNewModule_BuildsPasskeyHandlerWhenConfigured(t *testing.T) {
	t.Parallel()

	cfg := validModuleConfig()
	cfg.Credentials = stubCredentialStore{}
	cfg.Config.Passkey.Enabled = true
	cfg.Config.Passkey.RPDisplayName = "Example"
	cfg.Config.Passkey.RPID = "example.com"
	cfg.Config.Passkey.RPOrigins = []string{"https://example.com"}

	m, err := NewModule(cfg)
	if err != nil {
		t.Fatalf("NewModule error: %v", err)
	}
	if m.PasskeyHandler == nil {
		t.Fatal("PasskeyHandler is nil, want handler")
	}
}
