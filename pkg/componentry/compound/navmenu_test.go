package compound_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/go-sum/foundry/pkg/componentry/compound"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// renderNode renders a g.Node to a string for assertion.
func renderNode(t *testing.T, node g.Node) string {
	t.Helper()
	if node == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := node.Render(&buf); err != nil {
		t.Fatalf("renderNode: render failed: %v", err)
	}
	return buf.String()
}

// containsStr checks whether s contains substr.
func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}

// baseConfig builds a minimal NavConfig with a single section containing the provided items.
func baseConfig(items ...compound.NavItem) compound.NavConfig {
	return compound.NavConfig{
		Brand: compound.NavBrand{Label: "TestBrand", Href: "/"},
		Sections: []compound.NavSection{
			{Items: items},
		},
	}
}

// ---- NavMenu render tests ----

func TestNavMenu_RendersGuestItems(t *testing.T) {
	cfg := baseConfig(
		compound.NavItem{Visibility: compound.NavVisibilityGuest, Label: "Login", Href: "/login"},
		compound.NavItem{Visibility: compound.NavVisibilityUser, Label: "Profile", Href: "/profile"},
	)
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{
		Config:          cfg,
		IsAuthenticated: false,
	}))

	if !containsStr(got, "Login") {
		t.Errorf("expected 'Login' in guest output, got:\n%s", got)
	}
	if containsStr(got, "Profile") {
		t.Errorf("expected 'Profile' to be absent in guest output, got:\n%s", got)
	}
}

func TestNavMenu_RendersUserItems(t *testing.T) {
	cfg := baseConfig(
		compound.NavItem{Visibility: compound.NavVisibilityGuest, Label: "Login", Href: "/login"},
		compound.NavItem{Visibility: compound.NavVisibilityUser, Label: "Profile", Href: "/profile"},
	)
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{
		Config:          cfg,
		IsAuthenticated: true,
	}))

	if containsStr(got, "Login") {
		t.Errorf("expected 'Login' to be absent in authenticated output, got:\n%s", got)
	}
	if !containsStr(got, "Profile") {
		t.Errorf("expected 'Profile' in authenticated output, got:\n%s", got)
	}
}

func TestNavMenu_RendersBrand(t *testing.T) {
	cfg := compound.NavConfig{
		Brand: compound.NavBrand{Label: "Acme Corp", Href: "/home"},
	}
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{Config: cfg}))

	if !containsStr(got, "Acme Corp") {
		t.Errorf("expected brand label 'Acme Corp' in output, got:\n%s", got)
	}
	if !containsStr(got, `href="/home"`) {
		t.Errorf("expected brand href='/home' in output, got:\n%s", got)
	}
}

func TestNavMenu_RendersLink(t *testing.T) {
	cfg := baseConfig(
		compound.NavItem{Label: "About", Href: "/about"},
	)
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{Config: cfg}))

	if !containsStr(got, "About") {
		t.Errorf("expected 'About' in output, got:\n%s", got)
	}
	if !containsStr(got, `href="/about"`) {
		t.Errorf("expected href='/about' in output, got:\n%s", got)
	}
}

func TestNavMenu_RendersGroup(t *testing.T) {
	cfg := baseConfig(
		compound.NavItem{
			Label: "Products",
			Items: []compound.NavItem{
				{Label: "Widget A", Href: "/products/a"},
			},
		},
	)
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{Config: cfg}))

	if !containsStr(got, "<details") {
		t.Errorf("expected <details> element for group, got:\n%s", got)
	}
	if !containsStr(got, "Products") {
		t.Errorf("expected 'Products' label in group output, got:\n%s", got)
	}
	if !containsStr(got, "Widget A") {
		t.Errorf("expected child item 'Widget A' in group output, got:\n%s", got)
	}
}

func TestNavMenu_RendersForm(t *testing.T) {
	cfg := baseConfig(
		compound.NavItem{Label: "Sign out", Action: "/signout", Method: "post"},
	)
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{Config: cfg}))

	if !containsStr(got, `action="/signout"`) {
		t.Errorf("expected action='/signout' in output, got:\n%s", got)
	}
	if !containsStr(got, "<form") {
		t.Errorf("expected <form> element in output, got:\n%s", got)
	}
}

func TestNavMenu_ConfigEmbeddedSlots(t *testing.T) {
	cfg := compound.NavConfig{
		Brand: compound.NavBrand{Label: "Test", Href: "/"},
		Sections: []compound.NavSection{
			{Items: []compound.NavItem{{Slot: "theme"}}},
		},
		Slots: compound.NavSlots{
			"theme": compound.ControlSlot("Theme", h.Button(g.Text("Toggle"))),
		},
	}
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{Config: cfg}))
	if !containsStr(got, "Toggle") {
		t.Errorf("expected 'Toggle' from config-embedded slot, got:\n%s", got)
	}
}

func TestNavMenu_SlotMissing(t *testing.T) {
	cfg := baseConfig(
		compound.NavItem{Slot: "nonexistent"},
		compound.NavItem{Label: "Home", Href: "/"},
	)
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{
		Config: cfg,
		Slots:  compound.NavSlots{},
	}))

	// The slot item must be skipped; only "Home" appears
	if !containsStr(got, "Home") {
		t.Errorf("expected 'Home' link in output, got:\n%s", got)
	}
}

func TestNavMenu_SlotTextSlot(t *testing.T) {
	cfg := baseConfig(
		compound.NavItem{Slot: "username"},
	)
	slots := compound.NavSlots{
		"username": compound.TextSlot("Alice"),
	}
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{
		Config: cfg,
		Slots:  slots,
	}))

	if !containsStr(got, "Alice") {
		t.Errorf("expected 'Alice' from TextSlot in output, got:\n%s", got)
	}
}

func TestNavMenu_SlotControlSlot(t *testing.T) {
	cfg := baseConfig(
		compound.NavItem{Slot: "theme"},
	)
	control := h.Button(g.Text("Dark mode"))
	slots := compound.NavSlots{
		"theme": compound.ControlSlot("Theme", control),
	}
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{
		Config: cfg,
		Slots:  slots,
	}))

	if !containsStr(got, "Dark mode") {
		t.Errorf("expected 'Dark mode' button from ControlSlot in output, got:\n%s", got)
	}
}

func TestNavMenu_SlotFormSlot(t *testing.T) {
	cfg := baseConfig(
		compound.NavItem{Slot: "signout"},
	)
	slots := compound.NavSlots{
		"signout": compound.FormSlot(compound.FormSlotProps{
			Label:  "Sign out",
			Action: "/signout",
			Method: "post",
		}),
	}
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{
		Config: cfg,
		Slots:  slots,
	}))

	if !containsStr(got, "<form") {
		t.Errorf("expected <form> element from FormSlot in output, got:\n%s", got)
	}
	if !containsStr(got, `action="/signout"`) {
		t.Errorf("expected action='/signout' from FormSlot in output, got:\n%s", got)
	}
}

func TestNavMenu_CurrentPath(t *testing.T) {
	cfg := baseConfig(
		compound.NavItem{Label: "Dashboard", Href: "/dashboard"},
		compound.NavItem{Label: "Settings", Href: "/settings"},
	)
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{
		Config:      cfg,
		CurrentPath: "/dashboard",
	}))

	if !containsStr(got, `aria-current="page"`) {
		t.Errorf("expected aria-current=page for active link, got:\n%s", got)
	}
}

func TestNavMenu_CurrentPath_InactiveLink(t *testing.T) {
	cfg := baseConfig(
		compound.NavItem{Label: "Settings", Href: "/settings"},
	)
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{
		Config:      cfg,
		CurrentPath: "/dashboard",
	}))

	if containsStr(got, `aria-current="page"`) {
		t.Errorf("expected no aria-current for inactive link, got:\n%s", got)
	}
}

// ---- buildItem unit-level tests via NavMenu ----

func TestBuildItem_Separator(t *testing.T) {
	// A separator item should produce a separator element in mobile/drawer output.
	cfg := baseConfig(
		compound.NavItem{Label: "Home", Href: "/"},
		compound.NavItem{Type: "separator"},
		compound.NavItem{Label: "About", Href: "/about"},
	)
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{Config: cfg}))

	// NavSeparator renders role=separator in the mobile drawer
	if !containsStr(got, `role="separator"`) {
		t.Errorf("expected role=separator from separator item, got:\n%s", got)
	}
}

func TestBuildItem_EmptyReturnsNil(t *testing.T) {
	// A zero-value NavItem has no label, href, action, or items — it should be
	// silently dropped and not appear in the rendered output.
	cfg := baseConfig(
		compound.NavItem{},                           // empty — should be dropped
		compound.NavItem{Label: "Home", Href: "/"},   // should appear
	)
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{Config: cfg}))

	if !containsStr(got, "Home") {
		t.Errorf("expected 'Home' in output after empty item filtered, got:\n%s", got)
	}
}

// ---- TextSlot / ControlSlot / FormSlot zero-value guard tests ----

func TestTextSlot_EmptyTextReturnsEmptySlot(t *testing.T) {
	slot := compound.TextSlot("")
	// An empty slot has no render function — using it produces no extra content.
	cfg := baseConfig(compound.NavItem{Slot: "x"})
	slots := compound.NavSlots{"x": slot}
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{
		Config: cfg,
		Slots:  slots,
	}))
	// The slot exists but its render function is nil; NavNode still renders — it
	// just returns nil from Desktop/Mobile. Rendering must not panic.
	_ = got
}

func TestControlSlot_NilControlReturnsEmptySlot(t *testing.T) {
	slot := compound.ControlSlot("Label", nil)
	cfg := baseConfig(compound.NavItem{Slot: "ctrl"})
	slots := compound.NavSlots{"ctrl": slot}
	// Must not panic; slot is empty so the item renders with nil Desktop/Mobile.
	got := renderNode(t, compound.NavMenu(compound.NavMenuProps{
		Config: cfg,
		Slots:  slots,
	}))
	_ = got
}

// ---- NavItem validation tests ----

func TestNavItemValidation(t *testing.T) {
	v := validator.New()
	compound.RegisterNavValidation(v)

	tests := []struct {
		name    string
		item    compound.NavItem
		wantTag string // "" means expect no error
	}{
		{
			name:    "valid link",
			item:    compound.NavItem{Label: "Home", Href: "/"},
			wantTag: "",
		},
		{
			name:    "valid slot",
			item:    compound.NavItem{Slot: "x"},
			wantTag: "",
		},
		{
			name:    "valid separator",
			item:    compound.NavItem{Type: "separator"},
			wantTag: "",
		},
		{
			name:    "separator with label",
			item:    compound.NavItem{Type: "separator", Label: "x"},
			wantTag: "separator_only",
		},
		{
			name:    "slot with href",
			item:    compound.NavItem{Slot: "x", Href: "/foo"},
			wantTag: "slot_conflict",
		},
		{
			name:    "slot with action",
			item:    compound.NavItem{Slot: "x", Action: "/foo"},
			wantTag: "slot_conflict",
		},
		{
			name:    "matchprefix without href",
			item:    compound.NavItem{Label: "x", MatchPrefix: true},
			wantTag: "requires_href",
		},
		{
			name:    "method without action",
			item:    compound.NavItem{Label: "x", Method: "post"},
			wantTag: "requires_action",
		},
		{
			name: "hiddenfields without action",
			item: compound.NavItem{
				Label:        "x",
				HiddenFields: []compound.NavHiddenField{{Name: "n", Value: "v"}},
			},
			wantTag: "requires_action",
		},
		{
			name:    "href and action together",
			item:    compound.NavItem{Label: "x", Href: "/", Action: "/post"},
			wantTag: "conflicts_with_href",
		},
		{
			name:    "empty item no label href action or items",
			item:    compound.NavItem{},
			wantTag: "required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := v.Struct(tc.item)
			if tc.wantTag == "" {
				if err != nil {
					t.Errorf("expected no validation error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected validation error with tag %q, got nil", tc.wantTag)
			}
			var ve validator.ValidationErrors
			if !isValidationErrors(err, &ve) {
				t.Fatalf("expected validator.ValidationErrors, got: %T: %v", err, err)
			}
			if !hasTag(ve, tc.wantTag) {
				t.Errorf("expected ValidationErrors to contain tag %q, got tags: %s", tc.wantTag, collectTags(ve))
			}
		})
	}
}

func isValidationErrors(err error, out *validator.ValidationErrors) bool {
	ve, ok := err.(validator.ValidationErrors)
	if ok {
		*out = ve
	}
	return ok
}

func hasTag(ve validator.ValidationErrors, tag string) bool {
	for _, fe := range ve {
		if fe.Tag() == tag {
			return true
		}
	}
	return false
}

func collectTags(ve validator.ValidationErrors) string {
	tags := make([]string, 0, len(ve))
	for _, fe := range ve {
		tags = append(tags, fe.Tag())
	}
	return strings.Join(tags, ", ")
}
