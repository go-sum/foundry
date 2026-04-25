package app

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/jackc/pgx/v5/pgxpool"

	cfgpkg "github.com/go-sum/config"
	"github.com/go-sum/assets/publish"
	"github.com/go-sum/componentry/assets/iconset"
	"github.com/go-sum/componentry/icons"
	"github.com/go-sum/db"
	"github.com/go-sum/kv/redisstore"
	"github.com/go-sum/notification"
	"github.com/go-sum/notification/notifylog"
	"github.com/go-sum/queue"
	"github.com/go-sum/queue/pgstore"
	"github.com/go-sum/web"
	"github.com/go-sum/web/cookiecodec"
	"github.com/go-sum/web/otelweb"
	"github.com/go-sum/web/router"
	"github.com/go-sum/web/session"
	"github.com/go-sum/web/validate"

	config "github.com/go-sum/foundry/config"
	appdb "github.com/go-sum/foundry/db"
	"github.com/go-sum/foundry/internal/features/contact"
	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/foundry/internal/view/errorpage"
)

func provideRuntime(_ context.Context) (Runtime, error) {
	cfg, err := config.Load()
	if err != nil {
		return Runtime{}, err
	}
	return Runtime{
		Config: cfg,
		Logger: slog.Default(),
		Tracer: noop.NewTracerProvider().Tracer("app"),
	}, nil
}

func provideAssets(cfg *config.Config) (*publish.Manifest, *icons.Registry) {
	manifest := publish.Must(publish.New(cfg.Assets.PublicDir, cfg.Assets.URLPrefix))
	publish.RegisterSprites(iconset.Default.Sprites)
	publish.SetPathFunc(manifest.Path)

	iconReg := icons.NewRegistry()
	resolved := make(map[icons.Key]icons.Ref, len(iconset.Default.Icons))
	for key, ref := range iconset.Default.Icons {
		resolved[key] = icons.Ref{
			Sprite: publish.SpritePath(ref.Sprite),
			ID:     ref.ID,
		}
	}
	iconReg.RegisterSet(resolved)
	return manifest, iconReg
}

func provideSecurity(_ context.Context, runtime Runtime) (Security, session.Store, error) {
	cfg := runtime.Config

	origins := make([]string, 0, 1+len(cfg.Site.OriginAllowlist))
	if cfg.Site.BaseURL != "" {
		origins = append(origins, cfg.Site.BaseURL)
	}
	origins = append(origins, cfg.Site.OriginAllowlist...)

	sessCfg, store, err := provideSession(runtime)
	if err != nil {
		return Security{}, nil, err
	}

	sec := Security{
		CSRF:    cfg.CSRF,
		Headers: cfg.Headers,
		CSP:     cfg.CSP,
		Origins: origins,
		Session: sessCfg,
	}
	return sec, store, nil
}

func provideSession(runtime Runtime) (session.Config, session.Store, error) {
	var store session.Store
	switch runtime.Config.SessionStore {
	case "cookie":
		keyHex := cfgpkg.ExpandSecret("SECURITY_SESSION_KEY")
		if keyHex == "" {
			return session.Config{}, nil, config.ErrSessionKeyMissing
		}
		key, err := hex.DecodeString(keyHex)
		if err != nil {
			return session.Config{}, nil, fmt.Errorf("%w: %w", config.ErrSessionKeyInvalid, err)
		}
		codec, err := cookiecodec.New(cookiecodec.Config{
			Name:    runtime.Config.Session.CookieName,
			Secrets: [][]byte{key},
			Mode:    cookiecodec.AEAD,
		})
		if err != nil {
			return session.Config{}, nil, fmt.Errorf("session: cookie store: %w", err)
		}
		store = session.NewCookieStore(codec)
	default: // "memory"
		store = session.NewMemoryStore()
	}
	return session.NewConfig(runtime.Config.Session, store), store, nil
}

func connectWithRetry(ctx context.Context, name string, logger *slog.Logger, maxAttempts int, fn func() error) error {
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if attempt < maxAttempts {
			backoff := time.Duration(1<<attempt) * time.Second
			// ±25% jitter to prevent thundering herd on service restart.
			jitter := time.Duration(rand.Int64N(int64(backoff) / 2))
			if rand.IntN(2) == 0 {
				backoff += jitter
			} else {
				backoff -= jitter
			}
			logger.WarnContext(ctx, "service connection failed, retrying",
				"service", name,
				"attempt", attempt,
				"max_attempts", maxAttempts,
				"backoff", backoff,
				"error", err,
			)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return fmt.Errorf("%s: %w (context canceled during retry)", name, err)
			}
		}
	}
	return err
}

func provideServices(ctx context.Context, runtime Runtime, _ Security, rt *router.Router, pres Presentation) (Services, error) {
	// In testing environments, skip external service wiring.
	if runtime.Config.Env == config.Testing {
		return Services{}, nil
	}

	// DB
	var pool *pgxpool.Pool
	if err := connectWithRetry(ctx, "db", runtime.Logger, 3, func() error {
		var err error
		pool, err = db.Connect(ctx,
			db.WithProductionDefaults(),
			db.WithSlowQueryLogger(runtime.Logger, 500*time.Millisecond),
		)
		return err
	}); err != nil {
		return Services{}, fmt.Errorf("services: db: %w", err)
	}
	db.LogPoolStats(ctx, pool, runtime.Logger, 60*time.Second)

	// Schema registry — built from embedded schema.yaml; external schemas resolved by name.
	schemaReg, err := db.LoadRegistryFromYAML(appdb.ConfigYAML, appdb.SchemaFiles,
		db.WithResolver(appdb.ExternalSchemas()),
	)
	if err != nil {
		return Services{}, fmt.Errorf("services: schema registry: %w", err)
	}

	// Schema readiness — refuse to start if the database schema does not match this binary.
	if err := db.VerifyFingerprint(ctx, pool, schemaReg.Fingerprint()); err != nil {
		pool.Close()
		return Services{}, fmt.Errorf("services: schema not ready (run 'task db:migrate'): %w", err)
	}

	// KV
	kvStore := redisstore.New(redisstore.Config{Addr: runtime.Config.KV.Addr, Password: runtime.Config.KV.Password})
	if err := connectWithRetry(ctx, "kv", runtime.Logger, 3, func() error {
		return kvStore.Ping(ctx)
	}); err != nil {
		pool.Close()
		return Services{}, fmt.Errorf("services: kv: %w", err)
	}

	// Queue
	qStore := pgstore.New(pool)
	qDispatcher := queue.NewDispatcher(qStore, queue.WithDispatcherLogger(runtime.Logger))

	// Notification — log sender for dev/default; configurable in production
	senders := map[notification.Channel]notification.Sender{
		notification.ChannelLog: notifylog.New(runtime.Logger),
	}
	notifier := notification.NewDispatcher(senders, runtime.Logger)

	// Contact module
	contactMod := contact.NewModule(contact.ModuleConfig{
		Pool:      pool,
		KV:        kvStore,
		Queue:     qDispatcher,
		Notifier:  notifier,
		Router:    rt,
		Validator: validate.New(),
		Service: contact.ServiceConfig{
			RateLimit:  runtime.Config.Contact.RateLimit,
			RateWindow: runtime.Config.Contact.RateWindow,
			QueueName:  contact.QueueName,
		},
		Worker: contact.WorkerConfig{
			SendTo:   runtime.Config.Contact.SendTo,
			SendFrom: runtime.Config.Contact.SendFrom,
		},
		ViewOpts: pres.ViewOpts,
		Logger:   runtime.Logger,
	})

	// Queue processor
	processor := queue.NewProcessor(qStore, queue.WithLogger(runtime.Logger))
	processor.Register(contact.QueueName, contactMod.QueueHandler,
		queue.WithWorkers(2),
		queue.WithMaxAttempts(5),
		queue.WithTimeout(30*time.Second),
	)
	processor.Start(ctx)

	return Services{
		DBPool:         pool,
		KVStore:        kvStore,
		Queue:          qDispatcher,
		Processor:      processor,
		Notifier:       notifier,
		Contact:        contactMod,
		SchemaRegistry: schemaReg,
	}, nil
}

func provideErrorBoundary(runtime Runtime, routing *router.Router) web.Middleware {
	return web.ErrorBoundary(web.BoundaryConfig{
		Renderer:     &appErrorRenderer{rt: routing},
		Logger:       runtime.Logger,
		CaptureStack: runtime.Config.Env == config.Production,
		OnError:      otelweb.MakeOnError(),
		Op: func(c *web.Context) string {
			return c.Method() + " " + c.URL().Path
		},
		Subsystem: func(c *web.Context) string { return "http" },
		TraceID:   otelweb.ExtractTraceID(),
		SpanID:    otelweb.ExtractSpanID(),
		DedupeKey: func(c *web.Context) string {
			parts := strings.SplitN(c.URL().Path, "/", 3)
			if len(parts) > 1 {
				return c.Method() + "|" + parts[1]
			}
			return c.Method() + "|" + c.URL().Path
		},
	})
}

// appErrorRenderer implements web.ErrorRenderer for the starter application.
type appErrorRenderer struct {
	rt *router.Router
}

// RenderError renders the error as an HTML response, choosing full-page or
// HTMX fragment mode based on the request.
func (r *appErrorRenderer) RenderError(c *web.Context, e *web.Error) web.Response {
	vr := view.NewRequest(c)
	full := errorpage.ErrorPage(vr, e)
	partial := errorpage.ErrorContent(e)
	return view.RenderWithStatus(vr, e.Status, full, partial)
}
