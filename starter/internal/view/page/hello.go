package page

import (
	"github.com/go-sum/foundry/internal/view"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// HelloPage renders the hello page with a greeting for the given name.
func HelloPage(req view.Request, name string) g.Node {
	partial := HelloPartial(name)
	return req.Page("Hello "+name,
		h.Div(h.Class("max-w-2xl mx-auto py-16 px-4"),
			h.Div(h.ID("greeting"), partial),
			h.Div(h.Class("mt-8"),
				h.Label(h.Class("block text-sm font-medium text-gray-700 mb-2"),
					g.Text("Change name:"),
				),
				h.Input(
					h.Type("text"),
					h.Name("name"),
					h.Value(name),
					h.Class("border border-gray-300 rounded px-3 py-2"),
					g.Attr("hx-get", "/hello/greeting"),
					g.Attr("hx-trigger", "keyup changed delay:300ms"),
					g.Attr("hx-target", "#greeting"),
					g.Attr("hx-swap", "innerHTML"),
					g.Attr("hx-include", "this"),
				),
			),
			h.Div(h.Class("mt-4"),
				h.A(h.Href("/"), h.Class("text-blue-600 hover:underline"),
					g.Text("Back to home"),
				),
			),
		),
	)
}

// HelloPartial renders just the greeting fragment for HTMX swaps.
func HelloPartial(name string) g.Node {
	return h.Div(
		h.H1(h.Class("text-3xl font-bold text-gray-900 mb-4"),
			g.Textf("Hello, %s!", name),
		),
		h.P(h.Class("text-gray-600"),
			g.Text("This greeting was rendered server-side."),
		),
	)
}
