// Package layout provides the application's HTML shell.
package layout

import (
	"path/filepath"
	"strings"

	"github.com/go-sum/componentry/interactive/runtime"
	"github.com/go-sum/componentry/interactive/theme"
	"github.com/go-sum/componentry/ui/feedback"
	"github.com/go-sum/web/render"
	g "maragu.dev/gomponents"
	c "maragu.dev/gomponents/components"
	h "maragu.dev/gomponents/html"
)

// Props configures the full-page HTML shell.
type Props struct {
	Title          string
	Nonce          string
	CSRFToken      string
	CSRFFieldName  string
	CSRFHeaderName string
	Flash          []string
	Nav            g.Node              // optional navigation bar rendered above page content
	PathFunc       func(string) string // asset path resolver; nil falls back to bare /name
	Children       []g.Node
}

// Page renders a complete HTML5 document with locally served HTMX and Tailwind CSS.
func Page(p Props) g.Node {
	pathFn := p.PathFunc
	if pathFn == nil {
		pathFn = func(name string) string {
			return "/" + strings.TrimPrefix(filepath.ToSlash(name), "/")
		}
	}
	return c.HTML5(c.HTML5Props{
		Title:    p.Title,
		Language: "en",
		Head: []g.Node{
			h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1")),
			h.Meta(h.Name("csrf-token"), h.Content(p.CSRFToken)),
			render.HXCSRFMeta(render.CSRFProps{
				Token:      p.CSRFToken,
				FieldName:  p.CSRFFieldName,
				HeaderName: p.CSRFHeaderName,
			}),
			h.Link(h.Rel("stylesheet"), h.Href(pathFn("css/app.css"))),
			theme.InitScript(),
		},
		Body: []g.Node{
			h.Class("bg-background text-foreground min-h-screen flex flex-col"),
			h.Script(h.Src(pathFn("js/htmx.min.js")), h.Defer(), g.Attr("nonce", p.Nonce)),
			flashRegion(p.Flash),
			p.Nav,
			h.Main(h.Class("container mx-auto px-4 py-6 flex-1"),
				g.Group(p.Children),
			),
			runtime.ScriptSrc(pathFn("js/componentry.min.js")),
		},
	})
}

func flashRegion(messages []string) g.Node {
	nodes := make([]g.Node, len(messages))
	for i, msg := range messages {
		nodes[i] = feedback.Alert.Root(
			feedback.AlertProps{Variant: feedback.AlertDefault, Dismissible: true},
			feedback.Alert.Description(g.Text(msg)),
		)
	}
	return h.Div(
		h.ID("flash"),
		h.Class("container mx-auto px-4 pt-4 grid gap-2"),
		g.Attr("hx-swap-oob", "true"),
		g.Attr("aria-live", "polite"),
		g.Group(nodes),
	)
}
