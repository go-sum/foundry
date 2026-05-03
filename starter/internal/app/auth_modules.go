package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/auth/authui"
	"github.com/go-sum/foundry/pkg/auth/provider"
	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/notification/email"
	"github.com/go-sum/foundry/pkg/web/authn"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/validate"
)

func provideAuthModule(pc ProviderContext, store authStore, emailSender email.Sender) (*authn.Module, error) {
	tokenCodec, err := authn.NewTokenCodec(pc.Runtime.Config.Auth.TokenKeys)
	if err != nil {
		return nil, fmt.Errorf("auth: token codec: %w", err)
	}

	return authn.NewModule(authn.ModuleConfig{
		Router:          pc.Router,
		Validator:       pc.Validator,
		Logger:          pc.Runtime.Logger,
		Config:          pc.Runtime.Config.Auth.Identity,
		Users:           store,
		Credentials:     store,
		AdminUsers:      store,
		Notifier:        newEmailNotifier(emailSender, pc.Runtime.Config.App.Email.From),
		TokenCodec:      tokenCodec,
		TokenNonceStore: newKVNonceStore(pc.KVStore),
		Renderer:        authui.NewRenderer(authui.Config{Page: centeredAuthPageRenderer(pc.ViewOpts)}),
		AdminRenderer:   authui.NewAdminRenderer(authui.Config{Page: centeredAuthPageRenderer(pc.ViewOpts)}),
		AuthEntryPath:   router.NewResolver(pc.Router).Path("auth.connect"),
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
		ConsentRenderer: provider.NewStubConsentRenderer(),
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

type kvNonceStore struct {
	store kv.Store
}

func newKVNonceStore(s kv.Store) auth.TokenNonceStore {
	return &kvNonceStore{store: s}
}

func (n *kvNonceStore) HasConsumed(ctx context.Context, key string) (bool, error) {
	count, err := n.store.Exists(ctx, key)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (n *kvNonceStore) MarkConsumed(ctx context.Context, key string, ttl time.Duration) error {
	return n.store.Set(ctx, key, []byte("1"), kv.SetOptions{TTL: ttl})
}
