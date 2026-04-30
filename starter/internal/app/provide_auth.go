package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/auth/authui"
	authpgstore "github.com/go-sum/foundry/pkg/auth/pgstore"
	"github.com/go-sum/foundry/pkg/auth/provider"
	providerpgstore "github.com/go-sum/foundry/pkg/auth/provider/pgstore"
	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/authn"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/validate"
	viewstate "github.com/go-sum/foundry/pkg/web/viewstate"

	config "github.com/go-sum/foundry/config"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// kvNonceStore adapts kv.Store to the auth.TokenNonceStore interface.
type kvNonceStore struct {
	store kv.Store
}

func newKVNonceStore(s kv.Store) auth.TokenNonceStore {
	return &kvNonceStore{store: s}
}

func (k *kvNonceStore) HasConsumed(ctx context.Context, key string) (bool, error) {
	n, err := k.store.Exists(ctx, key)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (k *kvNonceStore) MarkConsumed(ctx context.Context, key string, ttl time.Duration) error {
	return k.store.Set(ctx, key, []byte("1"), kv.SetOptions{TTL: ttl})
}

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
	tokenCodec, err := authn.NewTokenCodec(cfg.Auth.TokenKeys)
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
			vr := viewstate.NewRequest(c, viewOpts...)
			centered := h.Div(
				h.Class("flex min-h-[calc(100vh-4rem)] items-center justify-center px-4"),
				h.Div(h.Class("w-full max-w-sm"), content),
			)
			return viewstate.Render(vr, vr.Page(title, centered), content)
		},
	}

	authMod, err := authn.NewModule(authn.ModuleConfig{
		Router:          rt,
		Validator:       val,
		Logger:          logger,
		Config:          cfg.Auth.Identity,
		Users:           authStore,
		Credentials:     authStore,
		AdminUsers:      authStore,
		Notifier:        notifier,
		TokenCodec:      tokenCodec,
		TokenNonceStore: newKVNonceStore(kvStore),
		Renderer:        authui.NewRenderer(uiCfg),
		AdminRenderer:   authui.NewAdminRenderer(uiCfg),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("auth: identity module: %w", err)
	}

	providerStore := providerpgstore.New(pool)
	providerMod, err := provider.NewProviderModule(provider.ProviderModuleConfig{
		Router:    rt,
		Validator: val,
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
		SigninPath:      router.NewResolver(rt).Path(authn.RouteSigninShow),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("auth: provider module: %w", err)
	}

	return authMod, providerMod, nil
}
