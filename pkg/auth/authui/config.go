// Package authui provides default auth UI renderers built with componentry.
// It implements auth.Renderer and auth.AdminRenderer using gomponents views
// and delegates page-level layout to the host application via PageFunc.
package authui

import (
	"github.com/go-sum/foundry/pkg/web"
	g "maragu.dev/gomponents"
)

// PageFunc renders content within the host application's page layout.
type PageFunc func(c *web.Context, title string, content g.Node) (web.Response, error)

// Config configures the auth UI renderers.
type Config struct {
	Page PageFunc
}
