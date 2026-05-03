package app

import (
	"context"

	"github.com/go-sum/foundry/internal/features/home"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/health"
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
	if svc.QueueStore != nil {
		checks = append(checks, home.Checker{
			Name: "Queue",
			Fn:   svc.QueueStore.Ping,
		})
	}
	return checks
}

func healthCheckers(svc Services) []health.Checker {
	var checkers []health.Checker
	if svc.DBPool != nil {
		checkers = append(checkers, health.DBChecker(svc.DBPool))
	}
	return checkers
}

func contactHandlers(svc Services) (web.Handler, web.Handler) {
	if svc.Contact != nil && svc.Contact.Handler != nil {
		return svc.Contact.Handler.Form, svc.Contact.Handler.Submit
	}
	return web.UnavailableHandler("contact"), web.UnavailableHandler("contact")
}
