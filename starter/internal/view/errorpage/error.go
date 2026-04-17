// Package errorpage provides full-page and partial error views.
package errorpage

import (
	"fmt"

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
		h.Div(h.Class("bg-white rounded-lg border border-gray-200 shadow-sm p-8"),
			// Status badge
			h.Div(h.Class("inline-flex items-center rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600 mb-4"),
				g.Text(fmt.Sprintf("%d", e.Status)),
			),
			// Title
			h.H1(h.Class("text-2xl font-bold text-gray-900 mb-3"),
				g.Text(e.Title),
			),
			// Public message or 5xx generic message
			errorMessage(e),
			// Back link
			h.Div(h.Class("mt-8"),
				h.A(h.Href("/"), h.Class("text-sm text-blue-600 hover:underline"),
					g.Text("Back to home"),
				),
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
			h.P(h.Class("text-sm text-gray-600 mb-2"),
				g.Text("Something went wrong. Please try again or contact support."),
			),
			instanceNote(e),
		})
	}
	msg := e.PublicMessage()
	if msg == "" || msg == e.Title {
		return nil
	}
	return h.P(h.Class("text-sm text-gray-600"),
		g.Text(msg),
	)
}

// instanceNote renders e.Instance in muted text when set. This aids support
// correlation without leaking any internal error detail.
func instanceNote(e *web.Error) g.Node {
	if e.Instance == "" {
		return nil
	}
	return h.P(h.Class("text-xs text-gray-400"),
		g.Textf("Reference: %s", e.Instance),
	)
}
