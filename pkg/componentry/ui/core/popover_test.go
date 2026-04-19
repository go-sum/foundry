package core_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/componentry/ui/core"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestPopover_Root(t *testing.T) {
	got := testutil.RenderNode(t, core.Popover.Root(core.PopoverRootProps{}))
	if !strings.Contains(got, `data-popover`) {
		t.Errorf("Popover.Root: expected data-popover attribute, got:\n%s", got)
	}
	// Default class applied when none provided
	if !strings.Contains(got, "relative inline-block") {
		t.Errorf("Popover.Root: expected default class 'relative inline-block', got:\n%s", got)
	}
}

func TestPopover_Root_customClass(t *testing.T) {
	got := testutil.RenderNode(t, core.Popover.Root(core.PopoverRootProps{Class: "my-custom-class"}))
	if !strings.Contains(got, "my-custom-class") {
		t.Errorf("Popover.Root custom class: expected 'my-custom-class', got:\n%s", got)
	}
	if strings.Contains(got, "relative inline-block") {
		t.Errorf("Popover.Root custom class: expected no default class when custom provided, got:\n%s", got)
	}
}

func TestPopover_Root_withID(t *testing.T) {
	got := testutil.RenderNode(t, core.Popover.Root(core.PopoverRootProps{ID: "my-popover"}))
	if !strings.Contains(got, `id="my-popover"`) {
		t.Errorf("Popover.Root with ID: expected id=my-popover, got:\n%s", got)
	}
}

func TestPopover_Trigger(t *testing.T) {
	got := testutil.RenderNode(t, core.Popover.Trigger(core.PopoverTriggerProps{},
		g.Text("Open"),
	))
	if !strings.Contains(got, "list-none") {
		t.Errorf("Popover.Trigger: expected list-none class, got:\n%s", got)
	}
	if !strings.Contains(got, "cursor-pointer") {
		t.Errorf("Popover.Trigger: expected cursor-pointer class, got:\n%s", got)
	}
	if !strings.Contains(got, "Open") {
		t.Errorf("Popover.Trigger: expected children 'Open', got:\n%s", got)
	}
}

func TestPopover_Trigger_customClass(t *testing.T) {
	got := testutil.RenderNode(t, core.Popover.Trigger(core.PopoverTriggerProps{Class: "extra-trigger"}))
	if !strings.Contains(got, "extra-trigger") {
		t.Errorf("Popover.Trigger custom class: expected 'extra-trigger', got:\n%s", got)
	}
	if !strings.Contains(got, "list-none") {
		t.Errorf("Popover.Trigger custom class: expected list-none still present, got:\n%s", got)
	}
}

func TestPopover_Content_defaultAlignment(t *testing.T) {
	got := testutil.RenderNode(t, core.Popover.Content(core.PopoverContentProps{}))
	// Default width
	if !strings.Contains(got, "w-72") {
		t.Errorf("Popover.Content default: expected w-72, got:\n%s", got)
	}
	// Default alignment: left-0
	if !strings.Contains(got, "left-0") {
		t.Errorf("Popover.Content default: expected left-0 alignment, got:\n%s", got)
	}
}

func TestPopover_Content_rightAlignment(t *testing.T) {
	got := testutil.RenderNode(t, core.Popover.Content(core.PopoverContentProps{Align: "right"}))
	if !strings.Contains(got, "right-0") {
		t.Errorf("Popover.Content right: expected right-0, got:\n%s", got)
	}
}

func TestPopover_Content_centerAlignment(t *testing.T) {
	got := testutil.RenderNode(t, core.Popover.Content(core.PopoverContentProps{Align: "center"}))
	if !strings.Contains(got, "left-1/2") {
		t.Errorf("Popover.Content center: expected left-1/2, got:\n%s", got)
	}
	if !strings.Contains(got, "-translate-x-1/2") {
		t.Errorf("Popover.Content center: expected -translate-x-1/2, got:\n%s", got)
	}
}

func TestPopover_Content_customWidth(t *testing.T) {
	got := testutil.RenderNode(t, core.Popover.Content(core.PopoverContentProps{Width: "w-96"}))
	if !strings.Contains(got, "w-96") {
		t.Errorf("Popover.Content custom width: expected w-96, got:\n%s", got)
	}
	if strings.Contains(got, "w-72") {
		t.Errorf("Popover.Content custom width: expected no w-72, got:\n%s", got)
	}
}
