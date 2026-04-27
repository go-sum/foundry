package form_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/foundry/pkg/componentry/form"
	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestFieldSet_minimal(t *testing.T) {
	got := testutil.RenderNode(t, form.FieldSet(form.FieldSetProps{}, g.Text("controls")))
	if !strings.HasPrefix(got, "<fieldset") {
		t.Errorf("FieldSet minimal: expected <fieldset> element, got:\n%s", got)
	}
	if strings.Contains(got, "<legend") {
		t.Errorf("FieldSet minimal: expected no legend without Legend prop, got:\n%s", got)
	}
	if !strings.Contains(got, "controls") {
		t.Errorf("FieldSet minimal: expected children 'controls', got:\n%s", got)
	}
}

func TestFieldSet_withLegend(t *testing.T) {
	got := testutil.RenderNode(t, form.FieldSet(form.FieldSetProps{
		Legend: "Shipping Address",
	}))
	if !strings.Contains(got, "<legend") {
		t.Errorf("FieldSet withLegend: expected <legend> element, got:\n%s", got)
	}
	if !strings.Contains(got, "Shipping Address") {
		t.Errorf("FieldSet withLegend: expected legend text, got:\n%s", got)
	}
}

func TestFieldSet_withDescription(t *testing.T) {
	got := testutil.RenderNode(t, form.FieldSet(form.FieldSetProps{
		ID:          "addr",
		Description: "Enter your shipping details",
	}))
	if !strings.Contains(got, "Enter your shipping details") {
		t.Errorf("FieldSet withDescription: expected description text, got:\n%s", got)
	}
	if !strings.Contains(got, "addr-description") {
		t.Errorf("FieldSet withDescription: expected ID 'addr-description', got:\n%s", got)
	}
	if !strings.Contains(got, `aria-describedby`) {
		t.Errorf("FieldSet withDescription: expected aria-describedby attr, got:\n%s", got)
	}
}

func TestFieldSet_withHint(t *testing.T) {
	got := testutil.RenderNode(t, form.FieldSet(form.FieldSetProps{
		ID:   "addr",
		Hint: "Required fields are marked *",
	}))
	if !strings.Contains(got, "Required fields are marked *") {
		t.Errorf("FieldSet withHint: expected hint text, got:\n%s", got)
	}
	if !strings.Contains(got, "addr-hint") {
		t.Errorf("FieldSet withHint: expected ID 'addr-hint', got:\n%s", got)
	}
}

func TestFieldSet_withErrors(t *testing.T) {
	got := testutil.RenderNode(t, form.FieldSet(form.FieldSetProps{
		ID:     "addr",
		Errors: []string{"Please fill in all required fields"},
	}))
	if !strings.Contains(got, "Please fill in all required fields") {
		t.Errorf("FieldSet withErrors: expected error text, got:\n%s", got)
	}
	if !strings.Contains(got, "addr-error") {
		t.Errorf("FieldSet withErrors: expected ID 'addr-error', got:\n%s", got)
	}
	if !strings.Contains(got, `aria-errormessage`) {
		t.Errorf("FieldSet withErrors: expected aria-errormessage attr, got:\n%s", got)
	}
}

func TestFieldSet_disabled(t *testing.T) {
	got := testutil.RenderNode(t, form.FieldSet(form.FieldSetProps{
		Disabled: true,
	}))
	if !strings.Contains(got, "disabled") {
		t.Errorf("FieldSet disabled: expected disabled attr, got:\n%s", got)
	}
}

func TestLegend(t *testing.T) {
	got := testutil.RenderNode(t, form.Legend(g.Text("Personal Info")))
	if !strings.HasPrefix(got, "<legend") {
		t.Errorf("Legend: expected <legend> element, got:\n%s", got)
	}
	if !strings.Contains(got, "text-sm font-medium") {
		t.Errorf("Legend: expected text-sm font-medium class, got:\n%s", got)
	}
	if !strings.Contains(got, "Personal Info") {
		t.Errorf("Legend: expected 'Personal Info' text, got:\n%s", got)
	}
}
