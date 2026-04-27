package feedback_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/go-sum/foundry/pkg/componentry/testutil"
	"github.com/go-sum/foundry/pkg/componentry/ui/feedback"
)

func TestSpinner_default(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Spinner(feedback.SpinnerProps{}))

	want := `<span><svg class="animate-spin size-5" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true"><circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" class="opacity-25"></circle><path fill="currentColor" class="opacity-75" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg></span>`
	if got != want {
		t.Errorf("Spinner default:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestSpinner_small(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Spinner(feedback.SpinnerProps{Size: feedback.SpinnerSm}))
	if !strings.Contains(got, `class="animate-spin size-4"`) {
		t.Errorf("Spinner small: expected size-4 class, got:\n%s", got)
	}
}

func TestSpinner_large(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Spinner(feedback.SpinnerProps{Size: feedback.SpinnerLg}))
	if !strings.Contains(got, `class="animate-spin size-8"`) {
		t.Errorf("Spinner large: expected size-8 class, got:\n%s", got)
	}
}

func TestSpinner_withLabel(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Spinner(feedback.SpinnerProps{Label: "Loading users"}))
	if !strings.Contains(got, `role="img"`) {
		t.Errorf("Spinner label: expected role=img, got:\n%s", got)
	}
	if !strings.Contains(got, `aria-label="Loading users"`) {
		t.Errorf("Spinner label: expected aria-label, got:\n%s", got)
	}
	if strings.Contains(got, `aria-hidden`) {
		t.Errorf("Spinner label: should not have aria-hidden when labelled, got:\n%s", got)
	}
}

func TestSpinner_withID(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Spinner(feedback.SpinnerProps{ID: "save-spinner"}))
	if !strings.Contains(got, `id="save-spinner"`) {
		t.Errorf("Spinner ID: expected id attribute, got:\n%s", got)
	}
}

func TestSpinner_htmxIndicator(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Spinner(feedback.SpinnerProps{
		ID:    "my-spinner",
		Extra: []g.Node{h.Class("htmx-indicator")},
	}))
	if !strings.Contains(got, `class="htmx-indicator"`) {
		t.Errorf("Spinner htmx: expected htmx-indicator class, got:\n%s", got)
	}
	if !strings.Contains(got, `id="my-spinner"`) {
		t.Errorf("Spinner htmx: expected id attribute, got:\n%s", got)
	}
}

func TestSpinner_decorativeAriaHidden(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Spinner(feedback.SpinnerProps{}))
	if !strings.Contains(got, `aria-hidden="true"`) {
		t.Errorf("Spinner decorative: expected aria-hidden=true, got:\n%s", got)
	}
}
