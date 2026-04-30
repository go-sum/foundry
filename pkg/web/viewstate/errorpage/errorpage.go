// Package errorpage provides full-page and partial error views.
package errorpage

import (
	"fmt"

	"github.com/go-sum/foundry/pkg/componentry/ui/core"
	"github.com/go-sum/foundry/pkg/componentry/ui/data"
	"github.com/go-sum/foundry/pkg/componentry/ui/feedback"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/viewstate"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ErrorPage wraps ErrorContent in the full-page layout.
func ErrorPage(req viewstate.Request, e *web.Error) g.Node {
	return req.Page(e.Title, ErrorContent(e))
}

// ErrorContent renders the error details as a self-contained card fragment.
// It never renders e.Cause or any internal detail. For 5xx errors it shows a
// generic retry message and, when set, e.Instance for support correlation.
func ErrorContent(e *web.Error) g.Node {
	return h.Div(h.Class("mx-auto flex max-w-2xl flex-col gap-6 py-16"),
		data.Card.Root(
			data.Card.Header(
				h.Div(
					h.Class("flex items-start justify-between gap-4"),
					h.Div(
						h.Class("space-y-1"),
						data.Card.Title(g.Text(e.Title)),
						data.Card.Description(g.Textf("%d %s", e.Status, e.Title)),
					),
					core.Badge(core.BadgeProps{
						Variant:  core.BadgeSecondary,
						Children: []g.Node{g.Text(fmt.Sprintf("HTTP %d", e.Status))},
					}),
				),
			),
			data.Card.Content(
				h.Div(
					h.Class("space-y-4"),
					feedback.Alert.Root(
						feedback.AlertProps{Variant: alertVariant(e.Status)},
						feedback.Alert.Description(g.Text(messageText(e))),
					),
					h.Div(
						h.Class("flex flex-wrap gap-3"),
						core.Button(core.ButtonProps{
							Label:   "Return Home",
							Href:    "/",
							Variant: core.VariantDefault,
						}),
					),
					instanceNote(e),
				),
			),
		),
	)
}

func alertVariant(status int) feedback.AlertVariant {
	if status >= 500 {
		return feedback.AlertDestructive
	}
	return feedback.AlertDefault
}

func messageText(e *web.Error) string {
	if e.Status >= 500 {
		return "Something went wrong. Please try again or contact support."
	}
	msg := e.PublicMessage()
	if msg == "" || msg == e.Title {
		return e.Title
	}
	return msg
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
