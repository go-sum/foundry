package app

import (
	"context"
	"net/http"

	"github.com/go-sum/foundry/internal/features/home"
	"github.com/go-sum/foundry/pkg/web"
)

func homeServiceChecks(svc Services) []home.Checker {
	var checks []home.Checker
	if svc.DBPool != nil {
		checks = append(checks, home.Checker{
			Name: "Database",
			Fn:   func(ctx context.Context) error { return svc.DBPool.Ping(ctx) },
		})
	}
	if svc.KVStore != nil {
		checks = append(checks, home.Checker{
			Name: "KV Store",
			Fn:   svc.KVStore.Ping,
		})
	}
	return checks
}

func healthCheckers(svc Services) []healthChecker {
	var checkers []healthChecker
	if svc.DBPool != nil {
		checkers = append(checkers, &dbHealthChecker{pool: svc.DBPool})
	}
	return checkers
}

func contactHandlers(svc Services) (web.Handler, web.Handler) {
	if svc.Contact != nil && svc.Contact.Handler != nil {
		return svc.Contact.Handler.Form, svc.Contact.Handler.Submit
	}
	return unavailableHandler("contact"), unavailableHandler("contact")
}

func unavailableHandler(feature string) web.Handler {
	return func(*web.Context) (web.Response, error) {
		return web.Response{}, web.ErrUnavailable(feature+" feature unavailable", nil)
	}
}

type healthChecker interface {
	Check(ctx context.Context) error
}

func healthHandler(checkers ...healthChecker) web.Handler {
	return func(c *web.Context) (web.Response, error) {
		for _, ch := range checkers {
			if err := ch.Check(c.Context()); err != nil {
				return web.Response{}, web.ErrUnavailable("database unhealthy", err)
			}
		}
		return web.Text(http.StatusOK, "ok"), nil
	}
}

type dbHealthChecker struct {
	pool interface {
		Ping(context.Context) error
	}
}

func (d *dbHealthChecker) Check(ctx context.Context) error {
	return d.pool.Ping(ctx)
}
