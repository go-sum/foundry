package provider

import (
	"fmt"
	"log/slog"

	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/validate"
)

// ProviderModule bundles the OAuth 2.0 authorization server handlers.
type ProviderModule struct {
	authorizeHandler *AuthorizeHandler
	tokenHandler     *TokenHandler
	discoveryHandler *DiscoveryHandler
	userinfoHandler  *UserinfoHandler

	signinPath func() string
	config     Config
}

// ProviderModuleConfig holds all external dependencies needed to wire the provider module.
type ProviderModuleConfig struct {
	Router    *router.Router
	Validator validate.Validator
	Logger    *slog.Logger

	Config Config

	Clients  ClientStore
	Codes    CodeStore
	Tokens   TokenStore
	Consents ConsentStore

	// Users is used by the userinfo endpoint to resolve user claims from a token.
	Users UserinfoUserReader

	// ConsentRenderer produces HTML for the consent screen.
	ConsentRenderer ConsentRenderer

	// SigninPath resolves the signin URL for redirecting unauthenticated users.
	SigninPath func() string
}

// NewProviderModule wires the OAuth 2.0 provider module.
func NewProviderModule(cfg ProviderModuleConfig) (*ProviderModule, error) {
	if cfg.Clients == nil {
		return nil, fmt.Errorf("provider: ClientStore is required")
	}
	if cfg.Codes == nil {
		return nil, fmt.Errorf("provider: CodeStore is required")
	}
	if cfg.Tokens == nil {
		return nil, fmt.Errorf("provider: TokenStore is required")
	}
	if cfg.Consents == nil {
		return nil, fmt.Errorf("provider: ConsentStore is required")
	}
	if cfg.Users == nil {
		return nil, fmt.Errorf("provider: Users (UserinfoUserReader) is required")
	}
	if cfg.ConsentRenderer == nil {
		return nil, fmt.Errorf("provider: ConsentRenderer is required")
	}
	if cfg.SigninPath == nil {
		return nil, fmt.Errorf("provider: SigninPath is required")
	}

	config := ApplyDefaults(cfg.Config)

	m := &ProviderModule{
		authorizeHandler: &AuthorizeHandler{
			clients:   cfg.Clients,
			codes:     cfg.Codes,
			consents:  cfg.Consents,
			renderer:  cfg.ConsentRenderer,
			config:    config,
			validator: cfg.Validator,
			logger:    cfg.Logger,
		},
		tokenHandler: &TokenHandler{
			clients:   cfg.Clients,
			codes:     cfg.Codes,
			tokens:    cfg.Tokens,
			config:    config,
			validator: cfg.Validator,
			logger:    cfg.Logger,
		},
		discoveryHandler: &DiscoveryHandler{
			config: cfg.Config,
			router: cfg.Router,
		},
		userinfoHandler: &UserinfoHandler{
			tokens: cfg.Tokens,
			users:  cfg.Users,
			logger: cfg.Logger,
		},
		signinPath: cfg.SigninPath,
		config:     config,
	}

	return m, nil
}
