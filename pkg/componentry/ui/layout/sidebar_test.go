package layout_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/foundry/pkg/componentry/ui/layout"
	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestSidebar(t *testing.T) {
	got := testutil.RenderNode(t, layout.Sidebar(layout.SidebarProps{}))
	// Hidden checkbox toggle control
	if !strings.Contains(got, `type="checkbox"`) {
		t.Errorf("Sidebar: expected checkbox input, got:\n%s", got)
	}
	if !strings.Contains(got, "sr-only") {
		t.Errorf("Sidebar: expected sr-only on checkbox, got:\n%s", got)
	}
	// Backdrop label
	if !strings.Contains(got, "bg-black/50") {
		t.Errorf("Sidebar: expected backdrop with bg-black/50, got:\n%s", got)
	}
	// Aside panel
	if !strings.Contains(got, "<aside") {
		t.Errorf("Sidebar: expected <aside> panel, got:\n%s", got)
	}
}

func TestSidebar_defaultID(t *testing.T) {
	got := testutil.RenderNode(t, layout.Sidebar(layout.SidebarProps{}))
	// default base id is "sidebar"
	if !strings.Contains(got, `id="sidebar-toggle"`) {
		t.Errorf("Sidebar default ID: expected id=sidebar-toggle, got:\n%s", got)
	}
	if !strings.Contains(got, `id="sidebar-panel"`) {
		t.Errorf("Sidebar default ID: expected id=sidebar-panel, got:\n%s", got)
	}
	if !strings.Contains(got, `id="sidebar-backdrop"`) {
		t.Errorf("Sidebar default ID: expected id=sidebar-backdrop, got:\n%s", got)
	}
}

func TestSidebar_customID(t *testing.T) {
	got := testutil.RenderNode(t, layout.Sidebar(layout.SidebarProps{ID: "nav-drawer"}))
	if !strings.Contains(got, `id="nav-drawer-toggle"`) {
		t.Errorf("Sidebar custom ID: expected id=nav-drawer-toggle, got:\n%s", got)
	}
	if !strings.Contains(got, `id="nav-drawer-panel"`) {
		t.Errorf("Sidebar custom ID: expected id=nav-drawer-panel, got:\n%s", got)
	}
	if !strings.Contains(got, `id="nav-drawer-backdrop"`) {
		t.Errorf("Sidebar custom ID: expected id=nav-drawer-backdrop, got:\n%s", got)
	}
}

func TestSidebar_navContent(t *testing.T) {
	navNode := g.Text("nav-content")
	got := testutil.RenderNode(t, layout.Sidebar(layout.SidebarProps{Nav: navNode}))
	if !strings.Contains(got, "nav-content") {
		t.Errorf("Sidebar nav: expected nav content in output, got:\n%s", got)
	}
}

func TestSidebar_ariaLabel(t *testing.T) {
	got := testutil.RenderNode(t, layout.Sidebar(layout.SidebarProps{}))
	if !strings.Contains(got, `aria-label="Mobile navigation"`) {
		t.Errorf("Sidebar: expected aria-label=Mobile navigation, got:\n%s", got)
	}
}

func TestToggleAttrs(t *testing.T) {
	attrs := layout.ToggleAttrs("my-nav")
	if len(attrs) == 0 {
		t.Fatal("ToggleAttrs: expected non-empty attrs")
	}
	// Render into a wrapper element to inspect
	got := testutil.RenderNode(t, g.El("label", attrs...))
	if !strings.Contains(got, `for="my-nav-toggle"`) {
		t.Errorf("ToggleAttrs: expected for=my-nav-toggle, got:\n%s", got)
	}
	if !strings.Contains(got, `aria-controls="my-nav-panel"`) {
		t.Errorf("ToggleAttrs: expected aria-controls=my-nav-panel, got:\n%s", got)
	}
}

func TestCloseAttrs(t *testing.T) {
	toggleAttrs := layout.ToggleAttrs("my-nav")
	closeAttrs := layout.CloseAttrs("my-nav")

	// Render both and compare — they must produce identical output.
	toggleGot := testutil.RenderNode(t, g.El("label", toggleAttrs...))
	closeGot := testutil.RenderNode(t, g.El("label", closeAttrs...))

	if toggleGot != closeGot {
		t.Errorf("CloseAttrs: expected same output as ToggleAttrs\n toggle: %s\n  close: %s", toggleGot, closeGot)
	}
}

func TestToggleAttrs_defaultID(t *testing.T) {
	got := testutil.RenderNode(t, g.El("label", layout.ToggleAttrs("")...))
	if !strings.Contains(got, `for="sidebar-toggle"`) {
		t.Errorf("ToggleAttrs default ID: expected for=sidebar-toggle, got:\n%s", got)
	}
}
