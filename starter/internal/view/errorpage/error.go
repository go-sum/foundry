// Package errorpage provides full-page and partial error views.
package errorpage

import (
	"fmt"

	"github.com/go-sum/componentry/ui/core"
	"github.com/go-sum/componentry/ui/data"
	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/web"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ErrorPage wraps ErrorContent in the full-page layout.
func ErrorPage(req view.Request, e *web.Error) g.Node {
	return req.Page(e.Title, ErrorContent(e))
}

// ErrorContent renders the error details as a self-contained card fragment.
// It never renders e.Cause or any internal detail. For 5xx errors it shows a
// generic retry message and, when set, e.Instance for support correlation.
func ErrorContent(e *web.Error) g.Node {
	return h.Div(h.Class("max-w-lg mx-auto py-24 px-4"),
		data.Card.Root(
			data.Card.Content(
				core.Badge(core.BadgeProps{
					Variant:  core.BadgeSecondary,
					Children: []g.Node{g.Text(fmt.Sprintf("%d", e.Status))},
				}),
				h.H1(h.Class("text-2xl font-bold text-card-foreground mt-4 mb-3"),
					g.Text(e.Title),
				),
				errorMessage(e),
			),
			data.Card.Footer(
				core.Button(core.ButtonProps{
					Href:    "/",
					Variant: core.VariantLink,
					Label:   "Back to home",
				}),
			),
		),
	)
}

// errorMessage renders the user-visible message section.
// For 5xx errors it always shows a generic retry message and never leaks cause.
// For 4xx errors it shows the public message when non-empty.
func errorMessage(e *web.Error) g.Node {
	if e.Status >= 500 {
		return g.Group([]g.Node{
			h.P(h.Class("text-sm text-muted-foreground mb-2"),
				g.Text("Something went wrong. Please try again or contact support."),
			),
			instanceNote(e),
		})
	}
	msg := e.PublicMessage()
	if msg == "" || msg == e.Title {
		return nil
	}
	return h.P(h.Class("text-sm text-muted-foreground"),
		g.Text(msg),
	)
}

// instanceNote renders e.Instance in muted text when set. This aids support
// correlation without leaking any internal error detail.
func instanceNote(e *web.Error) g.Node {
	if e.Instance == "" {
		return nil
	}
	return h.P(h.Class("text-xs text-muted-foreground"),
		g.Textf("Reference: %s", e.Instance),
	)
}
