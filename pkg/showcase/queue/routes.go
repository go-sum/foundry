package queue

import (
	"github.com/go-sum/web"
	"github.com/go-sum/web/router"
	"github.com/jackc/pgx/v5/pgxpool"
	g "maragu.dev/gomponents"
)

// PageFunc renders a full HTML page within the host application's layout.
// The showcase/queue package stays decoupled from starter/internal/view.
type PageFunc func(c *web.Context, title string, content g.Node) (web.Response, error)

// Config holds configuration for the queue panel.
type Config struct {
	BasePath   string
	Pool       *pgxpool.Pool
	Page       PageFunc
	PerPage    int
	MaxPerPage int
}

// DefaultConfig returns a Config with sensible defaults.
// Pool and Page must be set by the caller before passing to Routes.
func DefaultConfig() Config {
	return Config{
		BasePath:   "/showcase/queues",
		PerPage:    50,
		MaxPerPage: 500,
	}
}

// Routes returns the router nodes for the queue panel.
func Routes(cfg Config) []router.Node {
	h := newHandler(cfg)
	return []router.Node{
		router.GroupNode(cfg.BasePath,
			router.GET("", "queue.index", h.Index),
			router.GET("/{queue}", "queue.detail", h.Detail),
			router.GET("/{queue}/jobs", "queue.jobs", h.Jobs),
		),
	}
}
