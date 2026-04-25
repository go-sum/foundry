package db

import (
	"github.com/go-sum/web"
	"github.com/go-sum/web/router"
	"github.com/jackc/pgx/v5/pgxpool"
	g "maragu.dev/gomponents"
)

// PageFunc renders a full HTML page within the host application's layout.
// The showcase/db package stays decoupled from starter/internal/view.
type PageFunc func(c *web.Context, title string, content g.Node) (web.Response, error)

// Config holds configuration for the database panel.
type Config struct {
	BasePath   string
	Pool       *pgxpool.Pool
	Page       PageFunc
	Schema     string
	PerPage    int
	MaxPerPage int
}

// DefaultConfig returns a Config with sensible defaults.
// Pool and Page must be set by the caller before passing to Routes.
func DefaultConfig() Config {
	return Config{
		BasePath:   "/showcase/db",
		Schema:     "public",
		PerPage:    25,
		MaxPerPage: 100,
	}
}

// Routes returns the router nodes for the database panel.
func Routes(cfg Config) []router.Node {
	h := newHandler(cfg)
	return []router.Node{
		router.GroupNode(cfg.BasePath,
			router.GET("", "db.index", h.Index),
			router.GET("/tables/{table}", "db.table", h.Table),
			router.GET("/tables/{table}/data", "db.table.data", h.TableData),
		),
	}
}
