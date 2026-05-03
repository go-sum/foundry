package app

import (
	"fmt"

	authpgstore "github.com/go-sum/foundry/pkg/auth/pgstore"
	"github.com/go-sum/foundry/pkg/auth/provider"
	providerpgstore "github.com/go-sum/foundry/pkg/auth/provider/pgstore"
	"github.com/go-sum/foundry/pkg/notification/email"
	"github.com/go-sum/foundry/pkg/web/authn"
)

// provideAuth wires the auth identity module and the OAuth 2.0 Authorization Server.
func provideAuth(pc ProviderContext, sec Security, emailSender email.Sender) (*authn.Module, *provider.ProviderModule, error) {
	authStore := authpgstore.New(pc.Pool)
	authMod, err := provideAuthModule(pc, authStore, emailSender)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: identity module: %w", err)
	}

	providerStore := providerpgstore.New(pc.Pool)
	providerMod, err := provideOAuthProviderModule(pc.Runtime.Config.Auth.Provider, pc.Runtime.Logger, pc.Router, pc.Validator, providerStoreDeps{
		AuthStore:     authStore,
		ProviderStore: providerStore,
	}, authMod.RouteConfig().Signin.Name)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: provider module: %w", err)
	}

	return authMod, providerMod, nil
}
