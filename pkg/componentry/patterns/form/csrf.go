// Package form provides CSRF element builders and the Form validation interface.
package form

import (
	"cmp"
	"strings"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// CSRFProps carries CSRF token and optional field/header name overrides.
// Zero-value FieldName defaults to "_csrf"; zero-value HeaderName defaults to "X-CSRF-Token".
type CSRFProps struct {
	Token      string
	FieldName  string // defaults to "_csrf"
	HeaderName string // defaults to "X-CSRF-Token"
}

// CSRFField returns an HTML hidden input carrying the CSRF token.
// Pass secure.CSRFToken(c) as Token. FieldName defaults to "_csrf".
//
//	<input type="hidden" name="_csrf" value="<token>">
func CSRFField(p CSRFProps) g.Node {
	return h.Input(h.Type("hidden"), h.Name(cmp.Or(p.FieldName, "_csrf")), h.Value(p.Token))
}

// CSRFHeaders returns an hx-headers attribute that injects the CSRF token into
// all HTMX requests from the element and its children. Apply to <body> or the
// root HTMX container. HeaderName defaults to "X-CSRF-Token".
//
//	hx-headers="{\"X-CSRF-Token\":\"<token>\"}"
func CSRFHeaders(p CSRFProps) g.Node {
	name := cmp.Or(p.HeaderName, "X-CSRF-Token")
	return g.Attr("hx-headers", `{"`+escapeToken(name)+`":"`+escapeToken(p.Token)+`"}`)
}

// escapeToken escapes \ and " for safe embedding in a JSON string.
// CSRF tokens are base64url (A-Z a-z 0-9 - _ =) so this is defensive-only.
func escapeToken(token string) string {
	token = strings.ReplaceAll(token, `\`, `\\`)
	token = strings.ReplaceAll(token, `"`, `\"`)
	return token
}
