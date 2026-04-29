// Package layout provides structural shell components for page layout.
package layout

import (
	"cmp"
	"strings"

	icons "github.com/go-sum/foundry/pkg/componentry/icons"
	iconrender "github.com/go-sum/foundry/pkg/componentry/icons/render"
	"github.com/go-sum/foundry/pkg/componentry/ui/core"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// NavbarVisibility controls whether an item renders for guest, user, or all states.
type NavbarVisibility string

const (
	VisibilityAll   NavbarVisibility = "all"
	VisibilityGuest NavbarVisibility = "guest"
	VisibilityUser  NavbarVisibility = "user"
)

// NavbarSectionAlign controls where a section sits in the desktop and mobile layout.
type NavbarSectionAlign string

const (
	AlignStart NavbarSectionAlign = "start"
	AlignEnd   NavbarSectionAlign = "end"
)

// NavbarProps configures a responsive navigation shell.
type NavbarProps struct {
	ID              string
	Brand           NavbarBrand
	Sections        []NavbarSection
	IsAuthenticated bool
	CurrentPath     string
	Icons           *icons.Registry
}

// NavbarBrand configures the logo/wordmark shown at the start of the nav bar.
type NavbarBrand struct {
	Label    string
	Href     string
	LogoPath string
}

// NavbarSection groups related items and can be aligned to the leading or trailing edge.
type NavbarSection struct {
	Label string
	Align NavbarSectionAlign
	Items []NavbarItem
}

// NavbarItem is the typed item model rendered by Navbar.
type NavbarItem interface {
	navbarItem()
	render(navbarItemContext) g.Node
	visible(NavbarContext) bool
}

// NavbarContext carries request-state needed to render items consistently.
type NavbarContext struct {
	CurrentPath     string
	IsAuthenticated bool
}

// NavHiddenField renders a hidden input inside a NavForm.
type NavHiddenField struct {
	Name  string
	Value string
}

// NavLink renders a standard navigation link.
type NavLink struct {
	Visibility  NavbarVisibility
	Label       string
	Href        string
	Icon        icons.Key
	MatchPrefix bool
}

func (NavLink) navbarItem() {}

func (i NavLink) visible(ctx NavbarContext) bool {
	return visibilityMatches(i.Visibility, ctx.IsAuthenticated)
}

func (i NavLink) render(ctx navbarItemContext) g.Node {
	if i.Label == "" || i.Href == "" {
		return nil
	}
	current := linkIsCurrent(ctx.CurrentPath, i.Href, i.MatchPrefix)
	nodes := []g.Node{
		h.Href(i.Href),
		h.Class(navLinkClass(ctx.viewport, ctx.depth, current)),
	}
	if current {
		nodes = append(nodes, g.Attr("aria-current", "page"))
	}
	nodes = append(nodes, navItemContent(ctx.icons, i.Icon, i.Label)...)
	return h.A(nodes...)
}

// NavGroup renders a dropdown on desktop and an accordion section on mobile.
type NavGroup struct {
	Visibility  NavbarVisibility
	Label       string
	Href        string
	Icon        icons.Key
	MatchPrefix bool
	Items       []NavbarItem
}

func (NavGroup) navbarItem() {}

func (i NavGroup) visible(ctx NavbarContext) bool {
	return visibilityMatches(i.Visibility, ctx.IsAuthenticated)
}

func (i NavGroup) render(ctx navbarItemContext) g.Node {
	if i.Label == "" {
		return nil
	}
	nodes := submenuNodes(i.Href, i.Label, i.MatchPrefix, i.Icon, i.Items, ctx)
	if len(nodes) == 0 {
		return nil
	}
	if ctx.viewport == viewportDesktop && ctx.depth == 0 {
		return h.Details(
			h.Class("group relative flex items-stretch"),
			groupSummary(i.Label, i.Icon, ctx),
			h.Div(
				h.Class("absolute left-0 top-full z-50 mt-px flex min-w-[16rem] flex-col divide-y divide-border rounded-md border border-border bg-popover shadow-lg"),
				g.Group(nodes),
			),
		)
	}
	return h.Details(
		g.Attr("data-nav-group", string(ctx.viewport)),
		h.Class(groupContainerClass(ctx.depth)),
		groupSummary(i.Label, i.Icon, ctx),
		h.Div(
			h.Class(groupPanelClass(ctx.depth)),
			g.Group(nodes),
		),
	)
}

// NavSeparator renders a visual separator between related items.
type NavSeparator struct {
	Visibility NavbarVisibility
}

func (NavSeparator) navbarItem() {}

func (i NavSeparator) visible(ctx NavbarContext) bool {
	return visibilityMatches(i.Visibility, ctx.IsAuthenticated)
}

func (i NavSeparator) render(ctx navbarItemContext) g.Node {
	if ctx.viewport == viewportDesktop && ctx.depth == 0 {
		return nil
	}
	return navSeparator(ctx.depth)
}

// NavText renders non-interactive text, useful for user names or status labels.
type NavText struct {
	Visibility NavbarVisibility
	Text       string
	Icon       icons.Key
}

func (NavText) navbarItem() {}

func (i NavText) visible(ctx NavbarContext) bool {
	return visibilityMatches(i.Visibility, ctx.IsAuthenticated)
}

func (i NavText) render(ctx navbarItemContext) g.Node {
	if i.Text == "" {
		return nil
	}
	cls := navTextClass(ctx.viewport, ctx.depth)
	nodes := []g.Node{h.Class(cls)}
	nodes = append(nodes, navItemContent(ctx.icons, i.Icon, i.Text)...)
	return h.Span(nodes...)
}

// NavNode renders arbitrary caller-supplied nodes for desktop and mobile contexts.
type NavNode struct {
	Visibility NavbarVisibility
	Desktop    g.Node
	Mobile     g.Node
	RenderFn   func(NavbarContext, bool) g.Node
}

func (NavNode) navbarItem() {}

func (i NavNode) visible(ctx NavbarContext) bool {
	return visibilityMatches(i.Visibility, ctx.IsAuthenticated)
}

func (i NavNode) render(ctx navbarItemContext) g.Node {
	if i.RenderFn != nil {
		return i.RenderFn(ctx.NavbarContext, ctx.viewport == viewportDesktop)
	}
	if ctx.viewport == viewportDesktop {
		return i.Desktop
	}
	if i.Mobile != nil {
		return i.Mobile
	}
	return i.Desktop
}

// RenderNavText renders a NavText item with context-aware styling. Used by compound components.
func RenderNavText(text string, icon icons.Key, ctx NavbarContext, isDesktop bool, depth int) g.Node {
	vp := viewportMobile
	if isDesktop {
		vp = viewportDesktop
	}
	return NavText{Text: text, Icon: icon}.render(navbarItemContext{NavbarContext: ctx, viewport: vp, depth: depth})
}

// RenderNavForm renders a NavForm item with context-aware styling. Used by compound components.
func RenderNavForm(form NavForm, ctx NavbarContext, isDesktop bool, depth int) g.Node {
	vp := viewportMobile
	if isDesktop {
		vp = viewportDesktop
	}
	return form.render(navbarItemContext{NavbarContext: ctx, viewport: vp, depth: depth})
}

// NavForm renders an inline form action such as signout.
type NavForm struct {
	Visibility   NavbarVisibility
	Label        string
	Action       string
	Method       string
	Icon         icons.Key
	HiddenFields []NavHiddenField
}

func (NavForm) navbarItem() {}

func (i NavForm) visible(ctx NavbarContext) bool {
	return visibilityMatches(i.Visibility, ctx.IsAuthenticated)
}

func (i NavForm) render(ctx navbarItemContext) g.Node {
	if i.Action == "" {
		return nil
	}
	label := cmp.Or(i.Label, "Submit")
	method := cmp.Or(i.Method, "post")
	nodes := []g.Node{
		h.Method(method),
		h.Action(i.Action),
	}
	for _, field := range i.HiddenFields {
		if field.Name == "" {
			continue
		}
		nodes = append(nodes, h.Input(h.Type("hidden"), h.Name(field.Name), h.Value(field.Value)))
	}
	if ctx.viewport == viewportDesktop && ctx.depth == 0 {
		button := core.Button(core.ButtonProps{
			Variant:   core.VariantGhost,
			Size:      core.SizeSm,
			Type:      "submit",
			FullWidth: false,
			Children:  buttonChildren(ctx.icons, i.Icon, label),
		})
		nodes = append(nodes, button)
		return h.Form(nodes...)
	}
	buttonNodes := []g.Node{
		h.Type("submit"),
		h.Class(navActionClass(ctx.depth)),
	}
	buttonNodes = append(buttonNodes, navItemContent(ctx.icons, i.Icon, label)...)
	nodes = append(nodes, h.Button(buttonNodes...))
	return h.Form(nodes...)
}

type navbarViewport string

const (
	viewportDesktop navbarViewport = "desktop"
	viewportMobile  navbarViewport = "mobile"
)

type navbarItemContext struct {
	NavbarContext
	viewport navbarViewport
	depth    int
	icons    *icons.Registry
}

const defaultNavbarID = "navbar"

func navbarID(id string) string {
	return cmp.Or(id, defaultNavbarID)
}

// Navbar renders a CSS-first responsive navigation shell with dropdown groups on desktop and a drawer accordion on mobile.
func Navbar(p NavbarProps) g.Node {
	drawerID := navbarID(p.ID)

	return h.Nav(
		h.Class("w-full border-b md:border-b-0 bg-background text-foreground"),
		h.Div(
			h.Class("container mx-auto flex h-14 items-center px-4"),
			h.Div(h.Class("mr-4 flex shrink-0"), brandNode(p.Brand)),
			h.Div(
				h.Class("hidden min-w-0 flex-1 items-stretch self-stretch md:flex"),
				desktopNavRegion(p),
			),
			mobileToggleButton(drawerID),
		),
		g.If(len(p.Sections) > 0, h.Div(
			h.Class("md:hidden"),
			Sidebar(SidebarProps{
				ID:  drawerID,
				Nav: mobileDrawer(drawerID, p),
			}),
		)),
	)
}

func brandNode(brand NavbarBrand) g.Node {
	label := cmp.Or(brand.Label, "Starter")
	href := cmp.Or(brand.Href, "/")
	children := []g.Node{}
	if brand.LogoPath != "" {
		children = append(children, h.Img(
			h.Src(brand.LogoPath),
			h.Alt(label),
			h.Class("h-8 w-8 rounded-md object-contain"),
		))
	}
	children = append(children, h.Span(h.Class("truncate"), g.Text(label)))
	return h.A(
		h.Href(href),
		h.Class("flex shrink-0 items-center gap-3 text-lg font-semibold tracking-tight"),
		g.Group(children),
	)
}

func desktopNavRegion(p NavbarProps) g.Node {
	start := renderSections(viewportDesktop, p, AlignStart)
	end := renderSections(viewportDesktop, p, AlignEnd)
	return h.Div(
		h.Class("flex min-w-0 flex-1 items-stretch justify-between gap-6"),
		h.Div(h.Class("flex min-w-0 flex-1 items-stretch gap-6"), g.Group(start)),
		h.Div(h.Class("flex items-stretch gap-3"), g.Group(end)),
	)
}

func mobileDrawer(id string, p NavbarProps) g.Node {
	start := renderSections(viewportMobile, p, AlignStart)
	end := renderSections(viewportMobile, p, AlignEnd)
	children := []g.Node{
		h.Div(
			h.Class("flex items-center justify-between border-b border-border px-4 py-4"),
			brandNode(p.Brand),
			mobileCloseButton(id),
		),
		h.Div(
			h.Class("flex min-h-0 flex-1 flex-col gap-8 overflow-y-auto px-4 py-5"),
			h.Div(h.Class("flex flex-col gap-8"), g.Group(start)),
		),
	}
	if len(end) > 0 {
		children = append(children,
			h.Div(
				h.Class("border-t border-border px-4 py-5"),
				h.Div(h.Class("flex flex-col gap-6"), g.Group(end)),
			),
		)
	}
	return h.Div(
		h.Class("flex h-full flex-col"),
		g.Group(children),
	)
}

func renderSections(viewport navbarViewport, p NavbarProps, align NavbarSectionAlign) []g.Node {
	ctx := NavbarContext{CurrentPath: p.CurrentPath, IsAuthenticated: p.IsAuthenticated}
	nodes := make([]g.Node, 0, len(p.Sections))
	for _, section := range p.Sections {
		if sectionAlign(section.Align) != align {
			continue
		}
		node := renderSection(viewport, section, ctx, p.Icons)
		if node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func sectionAlign(align NavbarSectionAlign) NavbarSectionAlign {
	if align == AlignEnd {
		return AlignEnd
	}
	return AlignStart
}

func renderSection(viewport navbarViewport, section NavbarSection, ctx NavbarContext, r *icons.Registry) g.Node {
	items := renderItems(section.Items, navbarItemContext{NavbarContext: ctx, viewport: viewport, icons: r})
	if len(items) == 0 && section.Label == "" {
		return nil
	}
	if viewport == viewportDesktop {
		children := []g.Node{}
		if section.Label != "" {
			children = append(children, h.Span(
				h.Class("self-center text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"),
				g.Text(section.Label),
			))
		}
		children = append(children, h.Div(h.Class("flex items-stretch gap-2"), g.Group(items)))
		return h.Div(h.Class("flex min-w-0 items-stretch gap-3"), g.Group(children))
	}
	children := []g.Node{}
	if section.Label != "" {
		children = append(children, h.P(
			h.Class("px-1 text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"),
			g.Text(section.Label),
		))
	}
	children = append(children, h.Div(h.Class("w-full divide-y divide-border rounded-lg border border-border"), g.Group(items)))
	return h.Div(h.Class("flex flex-col gap-2"), g.Group(children))
}

func renderItems(items []NavbarItem, ctx navbarItemContext) []g.Node {
	nodes := make([]g.Node, 0, len(items))
	for _, item := range items {
		if item == nil || !item.visible(ctx.NavbarContext) {
			continue
		}
		node := item.render(ctx)
		if node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func submenuNodes(href, label string, matchPrefix bool, icon icons.Key, items []NavbarItem, ctx navbarItemContext) []g.Node {
	nodes := []g.Node{}
	if href != "" && label != "" {
		current := linkIsCurrent(ctx.CurrentPath, href, matchPrefix)
		linkNodes := []g.Node{
			h.Href(href),
			h.Class(navLinkClass(ctx.viewport, ctx.depth+1, current)),
		}
		if current {
			linkNodes = append(linkNodes, g.Attr("aria-current", "page"))
		}
		linkNodes = append(linkNodes, navItemContent(ctx.icons, icon, label)...)
		nodes = append(nodes, h.A(linkNodes...))
	}
	childNodes := renderItems(items, navbarItemContext{
		NavbarContext: ctx.NavbarContext,
		viewport:      ctx.viewport,
		depth:         ctx.depth + 1,
		icons:         ctx.icons,
	})
	if len(nodes) > 0 && len(childNodes) > 0 {
		nodes = append(nodes, navSeparator(ctx.depth+1))
	}
	return append(nodes, childNodes...)
}

func navIndent(depth int) string {
	switch {
	case depth >= 2:
		return "pl-12 pr-4 md:pl-8"
	case depth == 1:
		return "pl-8 pr-4 md:px-4"
	default:
		return "px-4"
	}
}

func navLinkClass(viewport navbarViewport, depth int, current bool) string {
	if viewport == viewportDesktop && depth == 0 {
		base := "inline-flex items-center gap-2 border-b-2 px-4 text-sm font-medium outline-none transition-colors focus-visible:ring-[3px] focus-visible:ring-ring/50"
		if current {
			return base + " border-primary text-foreground"
		}
		return base + " border-transparent text-muted-foreground hover:border-border hover:text-foreground"
	}
	indent := navIndent(depth)
	base := "outline-none transition-colors hover:bg-accent/60 hover:text-accent-foreground focus-visible:ring-[3px] focus-visible:ring-ring/50"
	if current {
		base += " bg-accent/60 text-accent-foreground"
	}
	return "block w-full " + indent + " py-4 md:py-3 text-sm font-medium " + base
}

func navTextClass(viewport navbarViewport, depth int) string {
	if viewport == viewportDesktop && depth == 0 {
		return "text-sm text-muted-foreground"
	}
	indent := navIndent(depth)
	return "block w-full " + indent + " py-4 md:py-3 text-sm font-medium text-foreground"
}

func navActionClass(depth int) string {
	indent := navIndent(depth)
	base := "w-full text-left outline-none transition-colors hover:bg-accent/60 hover:text-accent-foreground focus-visible:ring-[3px] focus-visible:ring-ring/50"
	return "block " + indent + " py-4 md:py-3 text-sm font-medium " + base
}

func groupSummary(label string, icon icons.Key, ctx navbarItemContext) g.Node {
	children := navItemContent(ctx.icons, icon, label)
	children = append(children, chevronIcon(ctx.icons))
	if ctx.viewport == viewportDesktop && ctx.depth == 0 {
		return h.Summary(
			h.Class("navmenu-summary flex list-none cursor-pointer items-center gap-2 border-b-2 border-transparent px-4 text-sm font-medium text-muted-foreground outline-none transition-colors hover:border-border hover:text-foreground focus-visible:ring-[3px] focus-visible:ring-ring/50"),
			g.Group(children),
		)
	}
	indent := navIndent(ctx.depth)
	return h.Summary(
		h.Class("navmenu-summary flex list-none cursor-pointer items-center justify-between gap-3 "+indent+" py-4 md:py-3 text-left text-sm font-medium outline-none transition-colors hover:bg-accent/60 hover:text-accent-foreground focus-visible:ring-[3px] focus-visible:ring-ring/50"),
		g.Group(children),
	)
}

func groupContainerClass(depth int) string {
	if depth > 0 {
		return "bg-background md:border-t md:border-border/70"
	}
	return "bg-background"
}

func groupPanelClass(depth int) string {
	base := "flex flex-col divide-y divide-border border-t"
	if depth > 0 {
		return base + " border-border md:border-border/70"
	}
	return base + " border-border"
}

func navSeparator(depth int) g.Node {
	return core.Separator(core.SeparatorProps{
		Extra: []g.Node{h.Class(separatorClass(depth))},
	})
}

func separatorClass(depth int) string {
	if depth > 0 {
		return "mx-2 my-2 md:mx-0"
	}
	return "mx-2 my-2"
}

func navItemContent(r *icons.Registry, icon icons.Key, label string) []g.Node {
	nodes := []g.Node{}
	if icon != "" {
		nodes = append(nodes, core.Icon(iconrender.PropsForRegistry(r, icon, core.IconProps{
			Size: "size-4 shrink-0 text-muted-foreground",
		})))
	}
	nodes = append(nodes, h.Span(g.Text(label)))
	return nodes
}

func buttonChildren(r *icons.Registry, icon icons.Key, label string) []g.Node {
	if icon == "" {
		return []g.Node{g.Text(label)}
	}
	return []g.Node{
		core.Icon(iconrender.PropsForRegistry(r, icon, core.IconProps{Size: "size-4 shrink-0"})),
		g.Text(label),
	}
}

func chevronIcon(r *icons.Registry) g.Node {
	return core.Icon(iconrender.PropsForRegistry(r, icons.ChevronDown, core.IconProps{
		Size: "navmenu-chevron size-4 shrink-0 text-muted-foreground transition-transform",
	}))
}

func linkIsCurrent(currentPath, href string, matchPrefix bool) bool {
	if currentPath == "" || href == "" {
		return false
	}
	if currentPath == href {
		return true
	}
	if !matchPrefix || href == "/" {
		return false
	}
	prefix := strings.TrimSuffix(href, "/")
	if prefix == "" {
		return false
	}
	return strings.HasPrefix(currentPath, prefix+"/")
}

func visibilityMatches(visibility NavbarVisibility, isAuthenticated bool) bool {
	switch visibility {
	case VisibilityGuest:
		return !isAuthenticated
	case VisibilityUser:
		return isAuthenticated
	default:
		return true
	}
}

func mobileToggleButton(id string) g.Node {
	nodes := append([]g.Node{
		h.Class("ml-auto inline-flex items-center justify-center rounded-md p-2 text-foreground transition-colors hover:bg-accent hover:text-accent-foreground md:hidden"),
	}, ToggleAttrs(id)...)
	nodes = append(nodes,
		h.Span(h.Class("sr-only"), g.Text("Open navigation menu")),
		h.Span(
			h.Class("inline-flex h-4 w-5 flex-col justify-between"),
			h.Span(h.Class("block h-0.5 w-full rounded-full bg-current")),
			h.Span(h.Class("block h-0.5 w-full rounded-full bg-current")),
			h.Span(h.Class("block h-0.5 w-full rounded-full bg-current")),
		),
	)
	return h.Label(nodes...)
}

func mobileCloseButton(id string) g.Node {
	nodes := append([]g.Node{
		h.Class("inline-flex items-center justify-center rounded-md p-2 text-foreground transition-colors hover:bg-accent hover:text-accent-foreground"),
	}, CloseAttrs(id)...)
	nodes = append(nodes,
		h.Span(h.Class("sr-only"), g.Text("Close navigation menu")),
		h.Span(
			h.Class("relative block size-4"),
			h.Span(h.Class("absolute left-1/2 top-1/2 block h-0.5 w-4 -translate-x-1/2 -translate-y-1/2 rotate-45 rounded-full bg-current")),
			h.Span(h.Class("absolute left-1/2 top-1/2 block h-0.5 w-4 -translate-x-1/2 -translate-y-1/2 -rotate-45 rounded-full bg-current")),
		),
	)
	return h.Label(nodes...)
}
