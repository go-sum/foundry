package page

import (
	"github.com/go-sum/componentry/form"
	"github.com/go-sum/componentry/ui/core"
	"github.com/go-sum/foundry/internal/view"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// HelloPage renders the hello page with a greeting for the given name.
func HelloPage(req view.Request, name, greetingURL, homeURL string) g.Node {
	partial := HelloPartial(name)
	return req.Page("Hello "+name,
		h.Div(h.Class("max-w-2xl mx-auto py-16 px-4"),
			h.Div(h.ID("greeting"), partial),
			h.Div(h.Class("mt-8"),
				form.Field(form.FieldProps{
					ID:    "name",
					Label: "Change name:",
					Control: form.Input(form.InputProps{
						ID:    "name",
						Name:  "name",
						Type:  form.TypeText,
						Value: name,
						Extra: []g.Node{
							g.Attr("hx-get", greetingURL),
							g.Attr("hx-trigger", "keyup changed delay:300ms"),
							g.Attr("hx-target", "#greeting"),
							g.Attr("hx-swap", "innerHTML"),
							g.Attr("hx-include", "this"),
						},
					}),
				}),
			),
			h.Div(h.Class("mt-4"),
				core.Button(core.ButtonProps{
					Href:    homeURL,
					Variant: core.VariantLink,
					Label:   "Back to home",
				}),
			),
		),
	)
}

// HelloPartial renders just the greeting fragment for HTMX swaps.
func HelloPartial(name string) g.Node {
	return h.Div(
		h.H1(h.Class("text-2xl font-bold text-foreground mb-4"),
			g.Textf("Hello, %s!", name),
		),
		h.P(h.Class("text-muted-foreground"),
			g.Text("This greeting was rendered server-side."),
		),
	)
}
