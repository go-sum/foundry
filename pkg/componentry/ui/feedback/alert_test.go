package feedback_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/componentry/ui/feedback"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestAlert(t *testing.T) {
	tests := []struct {
		name     string
		props    feedback.AlertProps
		children []g.Node
	}{
		{
			name:  "default variant",
			props: feedback.AlertProps{Variant: feedback.AlertDefault},
			children: []g.Node{
				feedback.Alert.Title(g.Text("Heads up")),
				feedback.Alert.Description(g.Text("This is an informational alert.")),
			},
		},
		{
			name:  "destructive variant",
			props: feedback.AlertProps{Variant: feedback.AlertDestructive},
			children: []g.Node{
				feedback.Alert.Title(g.Text("Error")),
				feedback.Alert.Description(g.Text("Something went wrong.")),
			},
		},
		{
			name:  "default dismissible",
			props: feedback.AlertProps{Variant: feedback.AlertDefault, Dismissible: true},
			children: []g.Node{
				feedback.Alert.Description(g.Text("Dismissible message.")),
			},
		},
		{
			name:  "destructive dismissible",
			props: feedback.AlertProps{Variant: feedback.AlertDestructive, Dismissible: true},
			children: []g.Node{
				feedback.Alert.Description(g.Text("Dismissible error.")),
			},
		},
		{
			name:  "with icon",
			props: feedback.AlertProps{Variant: feedback.AlertDefault, Icon: g.Text("ℹ")},
			children: []g.Node{
				feedback.Alert.Title(g.Text("Info")),
				feedback.Alert.Description(g.Text("Alert with icon column.")),
			},
		},
		{
			name:  "destructive with icon",
			props: feedback.AlertProps{Variant: feedback.AlertDestructive, Icon: g.Text("✕")},
			children: []g.Node{
				feedback.Alert.Title(g.Text("Error")),
				feedback.Alert.Description(g.Text("Destructive with icon.")),
			},
		},
		{
			name:     "with id",
			props:    feedback.AlertProps{ID: "my-alert", Variant: feedback.AlertDefault},
			children: []g.Node{feedback.Alert.Description(g.Text("Identified alert."))},
		},
		{
			name:     "zero value",
			props:    feedback.AlertProps{},
			children: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := testutil.RenderNode(t, feedback.Alert.Root(tc.props, tc.children...))
			want := testutil.LoadGolden(t)
			testutil.AssertEqualHTML(t, want, got)
		})
	}
}

func TestAlertTitle(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Alert.Title(g.Text("My Title")))
	want := `<h5 class="line-clamp-1 min-h-4 font-medium tracking-tight">My Title</h5>`
	if got != want {
		t.Errorf("AlertTitle mismatch:\n want: %s\n  got: %s", want, got)
	}
}

func TestAlertDescription(t *testing.T) {
	got := testutil.RenderNode(t, feedback.Alert.Description(g.Text("Body text.")))
	want := `<div class="grid justify-items-start gap-1 text-sm" data-alert-description="">Body text.</div>`
	if got != want {
		t.Errorf("AlertDescription mismatch:\n want: %s\n  got: %s", want, got)
	}
}

func TestAlertAriaLive(t *testing.T) {
	t.Run("default is polite", func(t *testing.T) {
		got := testutil.RenderNode(t, feedback.Alert.Root(
			feedback.AlertProps{Variant: feedback.AlertDefault},
			feedback.Alert.Description(g.Text("msg")),
		))
		if want := `aria-live="polite"`; !contains(got, want) {
			t.Errorf("expected %q in output, got: %s", want, got)
		}
	})

	t.Run("destructive is assertive", func(t *testing.T) {
		got := testutil.RenderNode(t, feedback.Alert.Root(
			feedback.AlertProps{Variant: feedback.AlertDestructive},
			feedback.Alert.Description(g.Text("error")),
		))
		if want := `aria-live="assertive"`; !contains(got, want) {
			t.Errorf("expected %q in output, got: %s", want, got)
		}
	})
}

func TestAlertList(t *testing.T) {
	t.Run("renders multiple alerts", func(t *testing.T) {
		got := testutil.RenderNode(t, feedback.Alert.List(
			[]string{"default", "error"},
			[]string{"First message", "Error message"},
		))
		want := testutil.LoadGolden(t)
		testutil.AssertEqualHTML(t, want, got)
	})

	t.Run("empty slices render empty", func(t *testing.T) {
		got := testutil.RenderNode(t, feedback.Alert.List(nil, nil))
		if got != "" {
			t.Errorf("expected empty output, got: %q", got)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := range len(s) - len(substr) + 1 {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
