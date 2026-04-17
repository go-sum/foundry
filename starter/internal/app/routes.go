package app

import (
	"net/http"

	"github.com/go-sum/foundry/internal/features/hello"
	"github.com/go-sum/foundry/internal/features/home"
	"github.com/go-sum/web"
	"github.com/go-sum/web/router"
)

// RegisterRoutes registers all application routes on the router.
func RegisterRoutes(r *router.Router) {
	homeH := home.NewHandler(r.Routes)
	helloH := hello.NewHandler(r.Routes)

	r.GET("/healthz", "health.check", func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	})
	r.GET("/", "home.show", homeH.Show)
	r.GET("/hello/greeting", "hello.greeting", helloH.Greeting)
	r.GET("/hello/{name}", "hello.show", helloH.Show)
}
