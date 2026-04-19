package core_test

import (
	"strings"
	"testing"

	"github.com/go-sum/componentry/ui/core"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestSeparator_horizontal(t *testing.T) {
	got := testutil.RenderNode(t, core.Separator(core.SeparatorProps{
		Orientation: core.OrientationHorizontal,
	}))
	if !strings.Contains(got, `role="separator"`) {
		t.Errorf("horizontal: expected role=separator, got:\n%s", got)
	}
	if !strings.Contains(got, `aria-orientation="horizontal"`) {
		t.Errorf("horizontal: expected aria-orientation=horizontal, got:\n%s", got)
	}
}

func TestSeparator_vertical(t *testing.T) {
	got := testutil.RenderNode(t, core.Separator(core.SeparatorProps{
		Orientation: core.OrientationVertical,
	}))
	if !strings.Contains(got, `role="separator"`) {
		t.Errorf("vertical: expected role=separator, got:\n%s", got)
	}
	if !strings.Contains(got, `aria-orientation="vertical"`) {
		t.Errorf("vertical: expected aria-orientation=vertical, got:\n%s", got)
	}
}

func TestSeparator_withLabel(t *testing.T) {
	got := testutil.RenderNode(t, core.Separator(core.SeparatorProps{
		Orientation: core.OrientationHorizontal,
		Label:       "OR",
	}))
	if !strings.Contains(got, "OR") {
		t.Errorf("withLabel: expected label text 'OR', got:\n%s", got)
	}
}

func TestSeparator_noLabel(t *testing.T) {
	got := testutil.RenderNode(t, core.Separator(core.SeparatorProps{
		Orientation: core.OrientationHorizontal,
	}))
	// Without a label, the label span should not appear
	if strings.Contains(got, "relative mx-auto bg-background px-2") {
		t.Errorf("noLabel: unexpected label span in output:\n%s", got)
	}
}

func TestSeparator_dashed(t *testing.T) {
	got := testutil.RenderNode(t, core.Separator(core.SeparatorProps{
		Orientation: core.OrientationHorizontal,
		Decoration:  core.DecorationDashed,
	}))
	if !strings.Contains(got, "border-dashed") {
		t.Errorf("dashed: expected border-dashed class, got:\n%s", got)
	}
}

func TestSeparator_dotted(t *testing.T) {
	got := testutil.RenderNode(t, core.Separator(core.SeparatorProps{
		Orientation: core.OrientationHorizontal,
		Decoration:  core.DecorationDotted,
	}))
	if !strings.Contains(got, "border-dotted") {
		t.Errorf("dotted: expected border-dotted class, got:\n%s", got)
	}
}

func TestSeparator_defaultDecoration(t *testing.T) {
	got := testutil.RenderNode(t, core.Separator(core.SeparatorProps{
		Orientation: core.OrientationHorizontal,
		Decoration:  core.DecorationDefault,
	}))
	if strings.Contains(got, "border-dashed") || strings.Contains(got, "border-dotted") {
		t.Errorf("default decoration: expected no dashed/dotted class, got:\n%s", got)
	}
}
