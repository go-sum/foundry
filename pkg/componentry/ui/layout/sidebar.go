package layout

import (
	"cmp"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

const defaultSidebarID = "sidebar"

func sidebarBaseID(id string) string {
	return cmp.Or(id, defaultSidebarID)
}

func sidebarPanelID(id string) string {
	return sidebarBaseID(id) + "-panel"
}

func sidebarToggleID(id string) string {
	return sidebarBaseID(id) + "-toggle"
}

func sidebarBackdropID(id string) string {
	return sidebarBaseID(id) + "-backdrop"
}

// ToggleAttrs returns the attributes a label should carry to toggle the sidebar drawer.
func ToggleAttrs(id string) []g.Node {
	baseID := sidebarBaseID(id)
	return []g.Node{
		h.For(sidebarToggleID(baseID)),
		g.Attr("aria-controls", sidebarPanelID(baseID)),
	}
}

// CloseAttrs returns the attributes a label should carry to close the sidebar drawer.
func CloseAttrs(id string) []g.Node {
	return ToggleAttrs(id)
}

// SidebarProps configures a CSS-only drawer sidebar.
type SidebarProps struct {
	ID      string
	Nav     g.Node
	Content []g.Node
}

// Sidebar renders an off-canvas sidebar controlled by a hidden checkbox and labels.
func Sidebar(p SidebarProps) g.Node {
	baseID := sidebarBaseID(p.ID)
	return h.Div(
		h.Class("contents"),
		h.Input(
			h.ID(sidebarToggleID(baseID)),
			h.Type("checkbox"),
			h.Class("peer sr-only"),
			g.Attr("aria-hidden", "true"),
		),
		h.Label(
			h.ID(sidebarBackdropID(baseID)),
			h.For(sidebarToggleID(baseID)),
			h.Class("fixed inset-0 z-40 hidden bg-black/50 peer-checked:block"),
			g.Attr("aria-hidden", "true"),
		),
		h.Aside(
			h.ID(sidebarPanelID(baseID)),
			h.Class("fixed inset-y-0 left-0 z-50 flex w-80 max-w-[85vw] -translate-x-full transform flex-col border-r border-border bg-background text-foreground shadow-xl transition-transform duration-200 peer-checked:translate-x-0"),
			h.Nav(
				h.Class("flex min-h-0 flex-1 flex-col"),
				g.Attr("aria-label", "Mobile navigation"),
				p.Nav,
			),
		),
		g.Group(p.Content),
	)
}
