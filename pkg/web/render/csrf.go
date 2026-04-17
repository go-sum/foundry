package render

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// CSRFField returns an HTML hidden input element carrying the CSRF token.
// Pass secure.CSRFToken(c) as token.
//
//	<input type="hidden" name="_csrf" value="<token>">
func CSRFField(token string) g.Node {
	return h.Input(
		h.Type("hidden"),
		h.Name("_csrf"),
		h.Value(token),
	)
}
