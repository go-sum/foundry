package app

import (
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/auth/authui"
	authpgstore "github.com/go-sum/foundry/pkg/auth/pgstore"
	"github.com/go-sum/foundry/pkg/auth/provider"
	providerpgstore "github.com/go-sum/foundry/pkg/auth/provider/pgstore"
	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/validate"

	config "github.com/go-sum/foundry/config"
	"github.com/go-sum/foundry/internal/view"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// provideAuth wires the auth identity module and the OAuth 2.0 Authorization Server.
func provideAuth(
	cfg *config.Config,
	logger *slog.Logger,
	pool *pgxpool.Pool,
	kvStore kv.Store,
	rt *router.Router,
	viewOpts []view.RequestOption,
) (*auth.Module, *provider.ProviderModule, error) {
	tokenCodec, err := auth.NewTokenCodec(cfg.Auth.TokenKeys)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: token codec: %w", err)
	}

	notifier, err := mustNotProductionLogNotifier(cfg.Env, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: notifier: %w", err)
	}

	authStore := authpgstore.New(pool)
	uiCfg := authui.Config{
		Page: func(c *web.Context, title string, content g.Node) (web.Response, error) {
			vr := view.NewRequest(c, viewOpts...)
			centered := h.Div(
				h.Class("flex min-h-[calc(100vh-4rem)] items-center justify-center px-4"),
				h.Div(h.Class("w-full max-w-sm"), content),
			)
			return view.Render(vr, vr.Page(title, centered), content)
		},
	}

	authMod, err := auth.NewModule(auth.ModuleConfig{
		Router:          rt,
		Validator:       validate.New(),
		Logger:          logger,
		Config:          cfg.Auth.Identity,
		Users:           authStore,
		Credentials:     authStore,
		AdminUsers:      authStore,
		Notifier:        notifier,
		TokenCodec:      tokenCodec,
		TokenNonceStore: auth.NewKVTokenNonceStore(kvStore),
		Renderer:        authui.NewRenderer(uiCfg),
		AdminRenderer:   authui.NewAdminRenderer(uiCfg),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("auth: identity module: %w", err)
	}

	providerStore := providerpgstore.New(pool)
	providerMod, err := provider.NewProviderModule(provider.ProviderModuleConfig{
		Router:    rt,
		Validator: validate.New(),
		Logger:    logger,
		Config: provider.Config{
			Issuer: cfg.Auth.Provider.Issuer,
		},
		Clients:         providerStore,
		Codes:           providerStore,
		Tokens:          providerStore,
		Consents:        providerStore,
		Users:           authStore,
		ConsentRenderer: stubConsentRenderer{},
		SigninPath:      router.NewResolver(rt).Path(auth.RouteSigninShow),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("auth: provider module: %w", err)
	}

	return authMod, providerMod, nil
}
