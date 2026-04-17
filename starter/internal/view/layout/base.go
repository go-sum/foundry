// Package layout provides the application's HTML shell.
package layout

import (
	g "maragu.dev/gomponents"
	c "maragu.dev/gomponents/components"
	h "maragu.dev/gomponents/html"
)

// Props configures the full-page HTML shell.
type Props struct {
	Title    string
	Children []g.Node
}

// Page renders a complete HTML5 document with locally served HTMX and Tailwind CSS.
func Page(p Props) g.Node {
	return c.HTML5(c.HTML5Props{
		Title:    p.Title,
		Language: "en",
		Head: []g.Node{
			h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1")),
			h.Link(h.Rel("stylesheet"), h.Href("/static/css/app.css")),
		},
		Body: []g.Node{
			h.Script(h.Src("/static/js/htmx.min.js"), h.Defer()),
			h.Div(h.Class("min-h-screen bg-gray-50"),
				g.Group(p.Children),
			),
		},
	})
}
