package app

import (
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/authn"
	"github.com/go-sum/foundry/pkg/web/compress"
	"github.com/go-sum/foundry/pkg/web/etag"
	"github.com/go-sum/foundry/pkg/web/htmx"
	"github.com/go-sum/foundry/pkg/web/otelweb"
	"github.com/go-sum/foundry/pkg/web/router"
	"github.com/go-sum/foundry/pkg/web/secure"
	"github.com/go-sum/foundry/pkg/web/serve"
	"github.com/go-sum/foundry/pkg/web/session"
)

func coreMiddleware(rt *router.Router, runtime Runtime, security Security) ([]web.Middleware, error) {
	mw := []web.Middleware{
		web.AsyncContext(),
		otelweb.Middleware(runtime.Tracer),
		web.WithRequestID(),
		serve.AccessLogMiddleware(runtime.Logger),
		provideErrorBoundary(runtime, rt),
		secure.AllowedHosts(secure.AllowedHostsConfig{
			Hosts: security.AllowedHosts,
			Skipper: func(c *web.Context) bool {
				return c.URL() != nil && c.URL().Path == "/healthz"
			},
		}),
		secure.Headers(security.Headers),
		secure.CSPNonce(security.CSP),
	}
	mw = append(mw,
		session.Middleware(security.Session),
		htmx.VaryMiddleware(),
		authn.LoadSession(),
	)
	return mw, nil
}

func contentMiddleware(sec Security) []web.Middleware {
	return []web.Middleware{
		web.WithMaxBody(8 << 20),
		web.MethodOverride(web.MethodOverrideConfig{}),
		compress.Middleware(compress.Config{}),
		etag.Middleware(etag.Config{}),
		secure.OriginGuard(secure.OriginGuardConfig{
			TrustedOrigins: sec.Origins,
			ServerOrigin:   sec.ServerOrigin,
		}),
	}
}

func apiMiddleware(sec Security, tokenPath string) []web.Middleware {
	return []web.Middleware{
		secure.OriginGuard(secure.OriginGuardConfig{
			TrustedOrigins: sec.Origins,
			ServerOrigin:   sec.ServerOrigin,
			// The token endpoint is server-to-server; machine clients do not send
			// browser-origin headers. tokenPath comes from the module's RouteConfig,
			// so it stays in sync with the registered route without duplicating the string.
			Skipper: func(c *web.Context) bool {
				return c.URL() != nil && c.URL().Path == tokenPath
			},
		}),
	}
}
