package auth

import (
	"fmt"
	"log/slog"

	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/validate"
)

// Module bundles the auth feature's handlers and internal dependencies.
type Module struct {
	AuthHandler    *AuthHandler
	PasskeyHandler *PasskeyHandler
	AdminHandler   *AdminHandler

	signinPath func() string
	userReader UserReader
	config     Config
}

// ModuleConfig holds all external dependencies needed to wire the auth module.
type ModuleConfig struct {
	Router    *router.Router
	Validator validate.Validator
	Logger    *slog.Logger

	Config Config

	Users       UserWriter
	Credentials CredentialStore
	AdminUsers  AdminStore

	Notifier   Notifier
	TokenCodec TokenCodec

	Renderer      Renderer
	AdminRenderer AdminRenderer
}

// NewModule wires the auth module and returns the assembled Module.
// The caller registers routes via router.Register(rt, auth.Routes(m)...).
func NewModule(cfg ModuleConfig) (*Module, error) {
	config := ApplyDefaults(cfg.Config)
	if cfg.Router == nil {
		return nil, fmt.Errorf("auth: Router is required")
	}
	if cfg.Validator == nil {
		return nil, fmt.Errorf("auth: Validator is required")
	}
	if cfg.Users == nil {
		return nil, fmt.Errorf("auth: Users (UserWriter) is required")
	}
	if cfg.Renderer == nil {
		return nil, fmt.Errorf("auth: Renderer is required")
	}
	if cfg.AdminUsers == nil {
		return nil, fmt.Errorf("auth: AdminUsers (AdminStore) is required")
	}
	if cfg.AdminRenderer == nil {
		return nil, fmt.Errorf("auth: AdminRenderer is required")
	}
	if config.EmailTOTP.Enabled && cfg.TokenCodec == nil {
		return nil, fmt.Errorf("auth: TokenCodec is required when email TOTP is enabled")
	}
	if config.Passkey.Enabled && cfg.Credentials == nil {
		return nil, fmt.Errorf("auth: Credentials (CredentialStore) is required when passkeys are enabled")
	}

	authSvc := NewAuthService(AuthServiceConfig{
		Users:      cfg.Users,
		Notifier:   cfg.Notifier,
		TokenCodec: cfg.TokenCodec,
		EmailTOTP:  config.EmailTOTP,
	})

	authHandler := &AuthHandler{
		svc:       authSvc,
		router:    cfg.Router,
		validator: cfg.Validator,
		renderer:  cfg.Renderer,
		config:    config,
	}

	m := &Module{
		AuthHandler: authHandler,
		config:      config,
		userReader:  cfg.Users,
	}

	// Resolve signin path lazily so routes can be registered after module creation.
	res := router.NewResolver(cfg.Router)
	m.signinPath = res.Path(RouteSigninShow)

	if config.Passkey.Enabled {
		passkeySvc, err := NewPasskeyService(cfg.Users, cfg.Credentials, config.Passkey)
		if err != nil {
			return nil, fmt.Errorf("auth: passkey service: %w", err)
		}
		m.PasskeyHandler = &PasskeyHandler{
			svc:       passkeySvc,
			router:    cfg.Router,
			validator: cfg.Validator,
		}
	}

	adminSvc := NewAdminService(cfg.AdminUsers)
	m.AdminHandler = &AdminHandler{
		svc:       adminSvc,
		router:    cfg.Router,
		validator: cfg.Validator,
		renderer:  cfg.AdminRenderer,
	}

	return m, nil
}
