package app

import (
	"context"
	"fmt"

	"github.com/go-sum/foundry/internal/common"
	"github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/notification/email"
	"github.com/go-sum/foundry/pkg/queue"
	"github.com/go-sum/foundry/pkg/web/ratelimit"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/validate"

	config "github.com/go-sum/foundry/config"
	"github.com/go-sum/foundry/internal/features/contact"
	"github.com/go-sum/foundry/internal/features/oauthclient"
)

func provideWebServices(ctx context.Context, runtime Runtime, sec Security, rt *router.Router, pres Presentation, kvStore kv.Store, limiter *ratelimit.Limiter, val validate.Validator) (Services, error) {
	core, err := common.Provide(ctx, runtime.Logger, runtime.Config.DB.DSN, runtime.Config.App.Email)
	if err != nil {
		return Services{}, err
	}
	if kvStore == nil {
		core.DBPool.Close()
		return Services{}, fmt.Errorf("services: kv: missing shared store")
	}

	pc := ProviderContext{
		Runtime:   runtime,
		Pool:      core.DBPool,
		KVStore:   kvStore,
		Router:    rt,
		Validator: val,
		ViewOpts:  pres.ViewOpts,
	}

	contactMod := provideContactModule(pc, limiter, core.Queue, core.EmailSender, sec.RateLimitKey, runtime.Config.App.Contact)

	authMod, oauthProvider, err := provideAuth(pc, sec, core.EmailSender)
	if err != nil {
		return Services{}, err
	}
	oauthClientH := oauthclient.New(runtime.Config.Auth.OAuthClient)

	return Services{
		DBPool:         core.DBPool,
		KVStore:        kvStore,
		RateLimiter:    limiter,
		QueueStore:     core.QueueStore,
		Queue:          core.Queue,
		EmailSender:    core.EmailSender,
		Contact:        contactMod,
		Auth:           authMod,
		OAuthProvider:  oauthProvider,
		OAuthClient:    oauthClientH,
		SchemaRegistry: core.SchemaRegistry,
	}, nil
}

func provideContactModule(pc ProviderContext, limiter *ratelimit.Limiter, qDispatcher *queue.Dispatcher, emailSender email.Sender, rateLimitKey ratelimit.KeyFunc, cfg config.ContactConfig) *contact.Module {
	return contact.NewModule(contact.ModuleConfig{
		Pool:         pc.Pool,
		RateLimiter:  limiter,
		Queue:        qDispatcher,
		EmailSender:  emailSender,
		Router:       pc.Router,
		Validator:    pc.Validator,
		ClientIPFunc: rateLimitKey,
		Service: contact.ServiceConfig{
			RateLimitProfile: config.RateLimitContactSubmitEmail,
			QueueName:        contact.QueueName,
		},
		Worker: contact.WorkerConfig{
			SendTo:   cfg.SendTo,
			SendFrom: cfg.SendFrom,
		},
		ViewOpts: pc.ViewOpts,
		Logger:   pc.Runtime.Logger,
	})
}

