// Package layout provides the application's HTML shell.
package layout

import (
	"github.com/go-sum/web/render"
	g "maragu.dev/gomponents"
	c "maragu.dev/gomponents/components"
	h "maragu.dev/gomponents/html"
)

// Props configures the full-page HTML shell.
type Props struct {
	Title     string
	Nonce     string
	CSRFToken string
	Flash     []string
	Children  []g.Node
}

// Page renders a complete HTML5 document with locally served HTMX and Tailwind CSS.
func Page(p Props) g.Node {
	return c.HTML5(c.HTML5Props{
		Title:    p.Title,
		Language: "en",
		Head: []g.Node{
			h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1")),
			h.Meta(h.Name("csrf-token"), h.Content(p.CSRFToken)),
			render.HXCSRFMeta(p.CSRFToken),
			h.Link(h.Rel("stylesheet"), h.Href("/static/css/app.css")),
		},
		Body: []g.Node{
			h.Script(h.Src("/static/js/htmx.min.js"), h.Defer(), g.Attr("nonce", p.Nonce)),
			flashRegion(p.Flash),
			h.Div(h.Class("min-h-screen bg-gray-50"),
				g.Group(p.Children),
			),
		},
	})
}

func flashRegion(messages []string) g.Node {
	nodes := make([]g.Node, len(messages))
	for i, msg := range messages {
		nodes[i] = h.Div(g.Text(msg))
	}
	return h.Div(
		h.ID("flash"),
		g.Attr("hx-swap-oob", "true"),
		g.Group(nodes),
	)
}
