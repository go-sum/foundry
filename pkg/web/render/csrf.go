package render

import (
	"cmp"

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

func (p CSRFProps) fieldName() string  { return cmp.Or(p.FieldName, "_csrf") }
func (p CSRFProps) headerName() string { return cmp.Or(p.HeaderName, "X-CSRF-Token") }

// CSRFField returns an HTML hidden input element carrying the CSRF token.
// Pass secure.CSRFToken(c) as Token. FieldName defaults to "_csrf".
//
//	<input type="hidden" name="_csrf" value="<token>">
func CSRFField(p CSRFProps) g.Node {
	return h.Input(
		h.Type("hidden"),
		h.Name(p.fieldName()),
		h.Value(p.Token),
	)
}
