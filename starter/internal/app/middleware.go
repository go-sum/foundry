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

func installGlobalMiddleware(rt *router.Router, runtime Runtime, security Security) {
	rt.Use(globalMiddleware(rt, runtime, security)...)
}

func globalMiddleware(rt *router.Router, runtime Runtime, security Security) []web.Middleware {
	return []web.Middleware{
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
		session.Middleware(security.Session),
		secure.CSRF(security.CSRF),
		htmx.VaryMiddleware(),
		authn.LoadSession(),
	}
}

func browserRouteMiddleware(sec Security) []web.Middleware {
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

func packageRouteMiddleware(sec Security) []web.Middleware {
	return []web.Middleware{
		secure.OriginGuard(secure.OriginGuardConfig{
			TrustedOrigins: sec.Origins,
			ServerOrigin:   sec.ServerOrigin,
			// /oauth/token is a server-to-server endpoint; machine clients
			// do not send Sec-Fetch-Site headers.
			Skipper: func(c *web.Context) bool {
				return c.URL() != nil && c.URL().Path == "/oauth/token"
			},
		}),
	}
}
