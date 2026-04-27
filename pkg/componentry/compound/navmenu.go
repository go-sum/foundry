// Package compound provides higher-level components assembled from layout primitives.
package compound

import (
	"github.com/go-playground/validator/v10"
	icons "github.com/go-sum/foundry/pkg/componentry/icons"
	"github.com/go-sum/foundry/pkg/componentry/ui/layout"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// NavVisibility controls whether a nav item renders for guests, authenticated users, or all states.
type NavVisibility string

const (
	NavVisibilityAll   NavVisibility = "all"
	NavVisibilityGuest NavVisibility = "guest"
	NavVisibilityUser  NavVisibility = "user"
)

// NavAlign controls where a nav section sits in the layout.
type NavAlign string

const (
	NavAlignStart NavAlign = "start"
	NavAlignEnd   NavAlign = "end"
)

// Well-known slot names for NavItem.Slot references.
const (
	SlotThemeToggle = "theme_toggle"
)

// NavBrand configures the logo/wordmark shown at the start of the nav bar.
type NavBrand struct {
	Label    string
	Href     string
	LogoPath string
}

// NavConfig is the rendering-agnostic configuration for a NavMenu.
type NavConfig struct {
	Brand    NavBrand
	Sections []NavSection
	Slots    NavSlots
}

// NavSection groups related nav items and can be aligned to the leading or trailing edge.
type NavSection struct {
	Label string
	Align NavAlign
	Items []NavItem
}

// NavHiddenField is a hidden input field inside a NavForm.
type NavHiddenField struct {
	Name  string
	Value string
}

// NavItem is a single item in a nav section.
type NavItem struct {
	Type         string
	Slot         string
	Visibility   NavVisibility
	Label        string
	Href         string
	Action       string
	Method       string
	Icon         string
	MatchPrefix  bool
	HiddenFields []NavHiddenField
	Items        []NavItem
}

// NavSlot holds the desktop and mobile nodes (or a render function) for a named slot.
type NavSlot struct {
	Desktop g.Node
	Mobile  g.Node
	render  func(layout.NavbarContext, bool) g.Node
}

// NavSlots is a map of slot name to NavSlot.
type NavSlots map[string]NavSlot

// TextSlot returns a NavSlot that renders a plain text nav item.
func TextSlot(text string) NavSlot {
	if text == "" {
		return NavSlot{}
	}
	return NavSlot{
		render: func(ctx layout.NavbarContext, isDesktop bool) g.Node {
			return layout.RenderNavText(text, "", ctx, isDesktop, 0)
		},
	}
}

// ControlSlot returns a NavSlot that wraps an arbitrary control node for desktop and mobile.
func ControlSlot(label string, control g.Node) NavSlot {
	if control == nil {
		return NavSlot{}
	}
	return NavSlot{
		Desktop: control,
		Mobile: h.Div(
			h.Class("flex items-center justify-between px-4 py-4 transition-colors hover:bg-accent/60"),
			h.Span(h.Class("text-sm text-muted-foreground"), g.Text(label)),
			control,
		),
	}
}

// FormSlotProps configures a FormSlot.
type FormSlotProps struct {
	Label        string
	Action       string
	Method       string
	Icon         icons.Key
	HiddenFields []layout.NavHiddenField
}

// FormSlot returns a NavSlot that renders a form-based nav action.
func FormSlot(p FormSlotProps) NavSlot {
	form := layout.NavForm{
		Label:        p.Label,
		Action:       p.Action,
		Method:       p.Method,
		Icon:         p.Icon,
		HiddenFields: p.HiddenFields,
	}
	return NavSlot{
		render: func(ctx layout.NavbarContext, isDesktop bool) g.Node {
			return layout.RenderNavForm(form, ctx, isDesktop, 0)
		},
	}
}

// NavMenuProps configures a NavMenu component.
type NavMenuProps struct {
	ID              string
	Config          NavConfig
	Slots           NavSlots
	CurrentPath     string
	IsAuthenticated bool
	Icons           *icons.Registry
}

// NavMenu renders a full responsive navigation bar assembled from NavConfig and NavSlots.
// Slots are resolved from NavMenuProps.Slots first; if nil, NavConfig.Slots is used.
func NavMenu(p NavMenuProps) g.Node {
	slots := p.Slots
	if slots == nil {
		slots = p.Config.Slots
	}
	return layout.Navbar(layout.NavbarProps{
		ID:              p.ID,
		Brand:           mapBrand(p.Config.Brand),
		Sections:        buildSections(p.Config.Sections, slots),
		CurrentPath:     p.CurrentPath,
		IsAuthenticated: p.IsAuthenticated,
		Icons:           p.Icons,
	})
}

// ValidateSlots walks cfg and returns the names of any Slot references
// that are not present in cfg.Slots. An empty return means all slots are wired.
func ValidateSlots(cfg NavConfig) []string {
	var missing []string
	for _, section := range cfg.Sections {
		collectMissingSlots(section.Items, cfg.Slots, &missing)
	}
	return missing
}

func collectMissingSlots(items []NavItem, slots NavSlots, missing *[]string) {
	for _, item := range items {
		if item.Slot != "" {
			if _, ok := slots[item.Slot]; !ok {
				*missing = append(*missing, item.Slot)
			}
		}
		if len(item.Items) > 0 {
			collectMissingSlots(item.Items, slots, missing)
		}
	}
}

func buildSections(sections []NavSection, slots NavSlots) []layout.NavbarSection {
	built := make([]layout.NavbarSection, 0, len(sections))
	for _, section := range sections {
		built = append(built, layout.NavbarSection{
			Label: section.Label,
			Align: mapAlign(section.Align),
			Items: buildItems(section.Items, slots),
		})
	}
	return built
}

func buildItems(items []NavItem, slots NavSlots) []layout.NavbarItem {
	built := make([]layout.NavbarItem, 0, len(items))
	for _, item := range items {
		if node := buildItem(item, slots); node != nil {
			built = append(built, node)
		}
	}
	return built
}

func buildItem(item NavItem, slots NavSlots) layout.NavbarItem {
	if item.Type == "separator" {
		return layout.NavSeparator{Visibility: mapVisibility(item.Visibility)}
	}
	if item.Slot != "" {
		slot, ok := slots[item.Slot]
		if !ok {
			return nil
		}
		return layout.NavNode{
			Visibility: mapVisibility(item.Visibility),
			Desktop:    slot.Desktop,
			Mobile:     slot.Mobile,
			RenderFn:   slot.render,
		}
	}
	icon := icons.Key(item.Icon)
	if len(item.Items) > 0 {
		return layout.NavGroup{
			Visibility:  mapVisibility(item.Visibility),
			Label:       item.Label,
			Href:        item.Href,
			Icon:        icon,
			MatchPrefix: item.MatchPrefix,
			Items:       buildItems(item.Items, slots),
		}
	}
	if item.Action != "" {
		return layout.NavForm{
			Visibility:   mapVisibility(item.Visibility),
			Label:        item.Label,
			Action:       item.Action,
			Method:       item.Method,
			Icon:         icon,
			HiddenFields: mapHiddenFields(item.HiddenFields),
		}
	}
	if item.Href != "" {
		return layout.NavLink{
			Visibility:  mapVisibility(item.Visibility),
			Label:       item.Label,
			Href:        item.Href,
			Icon:        icon,
			MatchPrefix: item.MatchPrefix,
		}
	}
	if item.Label != "" {
		return layout.NavText{
			Visibility: mapVisibility(item.Visibility),
			Text:       item.Label,
			Icon:       icon,
		}
	}
	return nil
}

func mapBrand(b NavBrand) layout.NavbarBrand {
	return layout.NavbarBrand{Label: b.Label, Href: b.Href, LogoPath: b.LogoPath}
}

var _navVisibilityMap = map[NavVisibility]layout.NavbarVisibility{
	NavVisibilityGuest: layout.VisibilityGuest,
	NavVisibilityUser:  layout.VisibilityUser,
}

func mapVisibility(v NavVisibility) layout.NavbarVisibility {
	if vis, ok := _navVisibilityMap[v]; ok {
		return vis
	}
	return layout.VisibilityAll
}

func mapAlign(a NavAlign) layout.NavbarSectionAlign {
	if a == NavAlignEnd {
		return layout.AlignEnd
	}
	return layout.AlignStart
}

func mapHiddenFields(fields []NavHiddenField) []layout.NavHiddenField {
	if len(fields) == 0 {
		return nil
	}
	out := make([]layout.NavHiddenField, len(fields))
	for i, f := range fields {
		out[i] = layout.NavHiddenField{Name: f.Name, Value: f.Value}
	}
	return out
}

// RegisterNavValidation registers struct-level validation rules for NavItem.
func RegisterNavValidation(v *validator.Validate) {
	v.RegisterStructValidation(navItemStructValidation, NavItem{})
}

func navItemStructValidation(sl validator.StructLevel) {
	item := sl.Current().Interface().(NavItem)

	if item.Type == "separator" {
		reportIfSet(sl, item.Slot, "Slot", "slot", "separator_only")
		reportIfSet(sl, item.Label, "Label", "label", "separator_only")
		reportIfSet(sl, item.Href, "Href", "href", "separator_only")
		reportIfSet(sl, item.Action, "Action", "action", "separator_only")
		reportIfSet(sl, item.Method, "Method", "method", "separator_only")
		reportIfSet(sl, item.Icon, "Icon", "icon", "separator_only")
		reportIfTrue(sl, item.MatchPrefix, "MatchPrefix", "match_prefix", "separator_only")
		reportIfLen(sl, len(item.HiddenFields), "HiddenFields", "hidden_fields", "separator_only")
		reportIfLen(sl, len(item.Items), "Items", "items", "separator_only")
		return
	}
	if item.MatchPrefix && item.Href == "" {
		sl.ReportError(item.MatchPrefix, "MatchPrefix", "match_prefix", "requires_href", "")
	}
	if item.Method != "" && item.Action == "" {
		sl.ReportError(item.Method, "Method", "method", "requires_action", "")
	}
	if len(item.HiddenFields) > 0 && item.Action == "" {
		sl.ReportError(item.HiddenFields, "HiddenFields", "hidden_fields", "requires_action", "")
	}
	if item.Slot != "" {
		reportIfSet(sl, item.Href, "Href", "href", "slot_conflict")
		reportIfSet(sl, item.Action, "Action", "action", "slot_conflict")
		reportIfSet(sl, item.Method, "Method", "method", "slot_conflict")
		reportIfTrue(sl, item.MatchPrefix, "MatchPrefix", "match_prefix", "slot_conflict")
		reportIfLen(sl, len(item.HiddenFields), "HiddenFields", "hidden_fields", "slot_conflict")
		reportIfLen(sl, len(item.Items), "Items", "items", "slot_conflict")
		return
	}
	hasHref := item.Href != ""
	hasAction := item.Action != ""
	hasItems := len(item.Items) > 0
	if hasHref && hasAction {
		sl.ReportError(item.Action, "Action", "action", "conflicts_with_href", "")
	}
	if hasAction && hasItems {
		sl.ReportError(item.Action, "Action", "action", "conflicts_with_items", "")
	}
	if (hasHref || hasAction || hasItems) && item.Label == "" {
		sl.ReportError(item.Label, "Label", "label", "required_for_item", "")
	}
	if !hasHref && !hasAction && !hasItems && item.Label == "" {
		sl.ReportError(item.Label, "Label", "label", "required", "")
	}
}

func reportIfSet(sl validator.StructLevel, value string, fieldName, jsonName, tag string) {
	if value != "" {
		sl.ReportError(value, fieldName, jsonName, tag, "")
	}
}

func reportIfTrue(sl validator.StructLevel, value bool, fieldName, jsonName, tag string) {
	if value {
		sl.ReportError(value, fieldName, jsonName, tag, "")
	}
}

func reportIfLen(sl validator.StructLevel, n int, fieldName, jsonName, tag string) {
	if n > 0 {
		sl.ReportError(n, fieldName, jsonName, tag, "")
	}
}
