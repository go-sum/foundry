package form_test

import (
	"testing"

	g "maragu.dev/gomponents"

	pform "github.com/go-sum/componentry/patterns/form"
	testutil "github.com/go-sum/componentry/testutil"
	"github.com/go-sum/web/render"
)

func TestCSRFField(t *testing.T) {
	const token = "test-csrf-token-abc123"

	got := testutil.RenderNode(t, pform.CSRFField(token))
	want := testutil.RenderNode(t, render.CSRFField(token))

	if got != want {
		t.Errorf("CSRFField output differs from web/render.CSRFField:\n want: %s\n  got: %s", want, got)
	}
}

func TestCSRFField_ExactOutput(t *testing.T) {
	const token = "abc123"
	got := testutil.RenderNode(t, pform.CSRFField(token))
	want := `<input type="hidden" name="_csrf" value="abc123">`
	if got != want {
		t.Errorf("CSRFField exact mismatch:\n want: %s\n  got: %s", want, got)
	}
}

func TestCSRFHeaders(t *testing.T) {
	const token = "test-csrf-token-abc123"

	got := testutil.RenderNode(t, g.El("div", pform.CSRFHeaders(token)))
	want := testutil.RenderNode(t, g.El("div", render.HXCSRFHeaders(token)))

	if got != want {
		t.Errorf("CSRFHeaders output differs from web/render.HXCSRFHeaders:\n want: %s\n  got: %s", want, got)
	}
}

func TestCSRFHeaders_ExactOutput(t *testing.T) {
	const token = "mytoken"
	// HXCSRFHeaders produces: hx-headers="{\"X-CSRF-Token\":\"mytoken\"}"
	// Gomponents HTML-encodes the double-quotes to &#34; in attribute values.
	got := testutil.RenderNode(t, g.El("div", pform.CSRFHeaders(token)))
	want := `<div hx-headers="{&#34;X-CSRF-Token&#34;:&#34;mytoken&#34;}"></div>`
	if got != want {
		t.Errorf("CSRFHeaders exact mismatch:\n want: %s\n  got: %s", want, got)
	}
}
