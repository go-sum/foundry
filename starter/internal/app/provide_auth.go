package app

import (
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	authpgstore "github.com/go-sum/foundry/pkg/auth/pgstore"
	"github.com/go-sum/foundry/pkg/auth/provider"
	providerpgstore "github.com/go-sum/foundry/pkg/auth/provider/pgstore"
	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/web/authn"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/validate"
	viewstate "github.com/go-sum/foundry/pkg/web/viewstate"

	config "github.com/go-sum/foundry/config"
)

// provideAuth wires the auth identity module and the OAuth 2.0 Authorization Server.
func provideAuth(
	cfg *config.Config,
	logger *slog.Logger,
	pool *pgxpool.Pool,
	kvStore kv.Store,
	rt *router.Router,
	viewOpts []viewstate.RequestOption,
	val validate.Validator,
) (*authn.Module, *provider.ProviderModule, error) {
	authStore := authpgstore.New(pool)
	authMod, err := provideAuthModule(cfg, logger, rt, val, authStore, kvStore, viewOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: identity module: %w", err)
	}

	providerStore := providerpgstore.New(pool)
	providerMod, err := provideOAuthProviderModule(cfg.Auth.Provider, logger, rt, val, providerStoreDeps{
		AuthStore:     authStore,
		ProviderStore: providerStore,
	}, authMod.RouteConfig().Signin.Name)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: provider module: %w", err)
	}

	return authMod, providerMod, nil
}
