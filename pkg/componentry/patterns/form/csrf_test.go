package form_test

import (
	"testing"

	g "maragu.dev/gomponents"

	pform "github.com/go-sum/componentry/patterns/form"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestCSRFField_ExactOutput(t *testing.T) {
	const token = "abc123"
	got := testutil.RenderNode(t, pform.CSRFField(pform.CSRFProps{Token: token}))
	want := `<input type="hidden" name="_csrf" value="abc123">`
	if got != want {
		t.Errorf("CSRFField exact mismatch:\n want: %s\n  got: %s", want, got)
	}
}

func TestCSRFField_EmptyToken(t *testing.T) {
	got := testutil.RenderNode(t, pform.CSRFField(pform.CSRFProps{}))
	want := `<input type="hidden" name="_csrf" value="">`
	if got != want {
		t.Errorf("CSRFField empty token mismatch:\n want: %s\n  got: %s", want, got)
	}
}

func TestCSRFField_CustomFieldName(t *testing.T) {
	got := testutil.RenderNode(t, pform.CSRFField(pform.CSRFProps{Token: "t", FieldName: "my_csrf"}))
	want := `<input type="hidden" name="my_csrf" value="t">`
	if got != want {
		t.Errorf("CSRFField custom field name mismatch:\n want: %s\n  got: %s", want, got)
	}
}

func TestCSRFHeaders_ExactOutput(t *testing.T) {
	const token = "mytoken"
	// Gomponents HTML-encodes double-quotes to &#34; in attribute values.
	got := testutil.RenderNode(t, g.El("div", pform.CSRFHeaders(pform.CSRFProps{Token: token})))
	want := `<div hx-headers="{&#34;X-CSRF-Token&#34;:&#34;mytoken&#34;}"></div>`
	if got != want {
		t.Errorf("CSRFHeaders exact mismatch:\n want: %s\n  got: %s", want, got)
	}
}

func TestCSRFHeaders_EscapesQuote(t *testing.T) {
	got := testutil.RenderNode(t, g.El("div", pform.CSRFHeaders(pform.CSRFProps{Token: `tok"en`})))
	want := `<div hx-headers="{&#34;X-CSRF-Token&#34;:&#34;tok\&#34;en&#34;}"></div>`
	if got != want {
		t.Errorf("CSRFHeaders quote escaping mismatch:\n want: %s\n  got: %s", want, got)
	}
}

func TestCSRFHeaders_EscapesBackslash(t *testing.T) {
	got := testutil.RenderNode(t, g.El("div", pform.CSRFHeaders(pform.CSRFProps{Token: `tok\en`})))
	want := `<div hx-headers="{&#34;X-CSRF-Token&#34;:&#34;tok\\en&#34;}"></div>`
	if got != want {
		t.Errorf("CSRFHeaders backslash escaping mismatch:\n want: %s\n  got: %s", want, got)
	}
}

func TestCSRFHeaders_CustomHeaderName(t *testing.T) {
	got := testutil.RenderNode(t, g.El("div", pform.CSRFHeaders(pform.CSRFProps{Token: "t", HeaderName: "X-Custom"})))
	want := `<div hx-headers="{&#34;X-Custom&#34;:&#34;t&#34;}"></div>`
	if got != want {
		t.Errorf("CSRFHeaders custom header name mismatch:\n want: %s\n  got: %s", want, got)
	}
}
