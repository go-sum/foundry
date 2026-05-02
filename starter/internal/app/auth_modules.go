package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	config "github.com/go-sum/foundry/config"
	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/auth/authui"
	"github.com/go-sum/foundry/pkg/auth/provider"
	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/web/authn"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/validate"
	viewstate "github.com/go-sum/foundry/pkg/web/viewstate"
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

func provideAuthModule(cfg *config.Config, logger *slog.Logger, rt *router.Router, val validate.Validator, authStore authStore, kvStore kv.Store, viewOpts []viewstate.RequestOption) (*authn.Module, error) {
	tokenCodec, err := authn.NewTokenCodec(cfg.Auth.TokenKeys)
	if err != nil {
		return nil, fmt.Errorf("auth: token codec: %w", err)
	}

	notifier, err := mustNotProductionLogNotifier(cfg.Env, logger)
	if err != nil {
		return nil, fmt.Errorf("auth: notifier: %w", err)
	}

	return authn.NewModule(authn.ModuleConfig{
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
		Renderer:        authui.NewRenderer(authui.Config{Page: centeredAuthPageRenderer(viewOpts)}),
		AdminRenderer:   authui.NewAdminRenderer(authui.Config{Page: centeredAuthPageRenderer(viewOpts)}),
		AuthEntryPath:   router.NewResolver(rt).Path("auth.connect"),
	})
}

func provideOAuthProviderModule(cfg provider.Config, logger *slog.Logger, rt *router.Router, val validate.Validator, authStore providerStoreDeps, signinRoute string) (*provider.ProviderModule, error) {
	return provider.NewProviderModule(provider.ProviderModuleConfig{
		Router:          rt,
		Validator:       val,
		Logger:          logger,
		Config:          cfg,
		Clients:         authStore.ProviderStore,
		Codes:           authStore.ProviderStore,
		Tokens:          authStore.ProviderStore,
		Consents:        authStore.ProviderStore,
		Users:           authStore.AuthStore,
		ConsentRenderer: stubConsentRenderer{},
		SigninPath:      router.NewResolver(rt).Path(signinRoute),
	})
}

type authStore interface {
	auth.UserWriter
	auth.CredentialStore
	auth.AdminStore
	provider.UserinfoUserReader
}

type providerStoreDeps struct {
	AuthStore     provider.UserinfoUserReader
	ProviderStore interface {
		provider.ClientStore
		provider.CodeStore
		provider.TokenStore
		provider.ConsentStore
	}
}
