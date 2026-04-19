// Package form provides pass-through helpers that expose web/render CSRF
// primitives under the componentry import path.
// Views that import componentry/patterns/* need not directly depend on web/render.
package form

import (
	g "maragu.dev/gomponents"

	"github.com/go-sum/web/render"
)

// CSRFField returns an HTML hidden input element carrying the CSRF token.
// This is a thin pass-through to web/render.CSRFField.
func CSRFField(token string) g.Node {
	return render.CSRFField(token)
}

// CSRFHeaders returns an hx-headers attribute that injects the CSRF token into
// all HTMX requests from the element and its children.
// This is a thin pass-through to web/render.HXCSRFHeaders.
func CSRFHeaders(token string) g.Node {
	return render.HXCSRFHeaders(token)
}
