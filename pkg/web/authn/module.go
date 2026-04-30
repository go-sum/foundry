package authn

import (
	"fmt"
	"log/slog"

	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/validate"
)

// Module bundles the auth feature's handlers and internal dependencies.
type Module struct {
	AuthHandler    *AuthHandler
	PasskeyHandler *PasskeyHandler
	AdminHandler   *AdminHandler

	routes     RouteConfig
	signinPath func() string
	userReader auth.UserReader
	config     auth.Config
}

// ModuleConfig holds all external dependencies needed to wire the auth module.
type ModuleConfig struct {
	Router    *router.Router
	Validator validate.Validator
	Logger    *slog.Logger

	Config      auth.Config
	RouteConfig RouteConfig

	Users       auth.UserWriter
	Credentials auth.CredentialStore
	AdminUsers  auth.AdminStore

	Notifier        auth.Notifier
	TokenCodec      auth.TokenCodec
	TokenNonceStore auth.TokenNonceStore

	Renderer      Renderer
	AdminRenderer AdminRenderer

	// AuthEntryPath overrides where RequireAuth redirects unauthenticated users.
	// When nil, redirects to the default signin route.
	AuthEntryPath func() string

	// SignoutRedirectPath overrides where users are sent after signing out.
	// When nil, redirects to "/".
	SignoutRedirectPath func() string
}

// RequireAuth returns middleware that redirects unauthenticated requests to the
// configured auth entry path (signin or OAuth connect, depending on how the
// module was wired).
func (m *Module) RequireAuth() web.Middleware {
	return RequireAuth(m.signinPath)
}

// RouteConfig returns the resolved route configuration for this module.
func (m *Module) RouteConfig() RouteConfig {
	return m.routes
}

// NewModule wires the auth module and returns the assembled Module.
// The caller registers routes via router.Register(rt, authn.Routes(m)...).
func NewModule(cfg ModuleConfig) (*Module, error) {
	config := auth.ApplyDefaults(cfg.Config)
	routes := applyRouteDefaults(cfg.RouteConfig)
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
	if config.EmailTOTP.Enabled && cfg.TokenNonceStore == nil {
		return nil, fmt.Errorf("auth: TokenNonceStore is required when email TOTP is enabled")
	}
	if config.Passkey.Enabled && cfg.Credentials == nil {
		return nil, fmt.Errorf("auth: Credentials (CredentialStore) is required when passkeys are enabled")
	}

	authSvc := auth.NewAuthService(auth.AuthServiceConfig{
		Users:      cfg.Users,
		Notifier:   cfg.Notifier,
		TokenCodec: cfg.TokenCodec,
		NonceStore: cfg.TokenNonceStore,
		EmailTOTP:  config.EmailTOTP,
	})

	res := router.NewResolver(cfg.Router)
	signoutPath := func() string { return "/" }
	if cfg.SignoutRedirectPath != nil {
		signoutPath = cfg.SignoutRedirectPath
	}

	authHandler := &AuthHandler{
		svc:         authSvc,
		router:      cfg.Router,
		validator:   cfg.Validator,
		renderer:    cfg.Renderer,
		config:      config,
		signoutPath: signoutPath,
		routes:      routes,
	}

	m := &Module{
		AuthHandler: authHandler,
		config:      config,
		userReader:  cfg.Users,
		routes:      routes,
	}

	// Resolve signin path lazily so routes can be registered after module creation.
	m.signinPath = res.Path(routes.Signin.Name)
	if cfg.AuthEntryPath != nil {
		m.signinPath = cfg.AuthEntryPath
	}

	if config.Passkey.Enabled {
		passkeySvc, err := auth.NewPasskeyService(cfg.Users, cfg.Credentials, config.Passkey)
		if err != nil {
			return nil, fmt.Errorf("auth: passkey service: %w", err)
		}
		m.PasskeyHandler = &PasskeyHandler{
			svc:       passkeySvc,
			router:    cfg.Router,
			validator: cfg.Validator,
		}
	}

	adminSvc := auth.NewAdminService(cfg.AdminUsers)
	m.AdminHandler = &AdminHandler{
		svc:       adminSvc,
		router:    cfg.Router,
		validator: cfg.Validator,
		renderer:  cfg.AdminRenderer,
		routes:    routes,
	}

	return m, nil
}
