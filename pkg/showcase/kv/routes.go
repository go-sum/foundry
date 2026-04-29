package kv

import (
	kvstore "github.com/go-sum/foundry/pkg/kv"
	"github.com/go-sum/foundry/pkg/showcase/base"
	"github.com/go-sum/foundry/pkg/web/router"
)

// Config holds configuration for the KV panel.
type Config struct {
	BasePath   string
	Store      kvstore.Store
	Page       base.PageFunc
	PerPage    int
	MaxPerPage int
}

// DefaultConfig returns a Config with sensible defaults.
// Store and Page must be set by the caller before passing to Routes.
func DefaultConfig() Config {
	return Config{
		BasePath:   "/showcase/kv",
		PerPage:    50,
		MaxPerPage: 500,
	}
}

// Routes returns the router nodes for the KV panel.
func Routes(cfg Config) []router.Node {
	h := newHandler(cfg)
	return []router.Node{
		router.Group(cfg.BasePath,
			router.GET("", "kv.index", h.Index),
			router.GET("/keys/{key}", "kv.key", h.Key),
			router.GET("/keys/{key}/value", "kv.key.value", h.KeyValue),
		),
	}
}
