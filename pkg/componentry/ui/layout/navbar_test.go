package layout_test

import (
	"bytes"
	"strings"
	"testing"

	testutil "github.com/go-sum/componentry/testutil"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/go-sum/componentry/ui/layout"
)

// renderNodeLayout renders a g.Node to a string for assertion.
func renderNodeLayout(t *testing.T, node g.Node) string {
	t.Helper()
	if node == nil {
		t.Fatal("renderNodeLayout: node is nil")
	}
	var buf bytes.Buffer
	if err := node.Render(&buf); err != nil {
		t.Fatalf("renderNodeLayout: render failed: %v", err)
	}
	return buf.String()
}

// containsStr is a local helper to check for substring presence in HTML output.
func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestNavbar(t *testing.T) {
	tests := []struct {
		name  string
		props layout.NavbarProps
	}{
		{
			name: "brand only",
			props: layout.NavbarProps{
				Brand: layout.NavbarBrand{Label: "MyApp", Href: "/"},
			},
		},
		{
			name: "with links",
			props: layout.NavbarProps{
				Brand: layout.NavbarBrand{Label: "MyApp", Href: "/"},
				Sections: []layout.NavbarSection{
					{
						Items: []layout.NavbarItem{
							layout.NavLink{Label: "Home", Href: "/"},
							layout.NavLink{Label: "About", Href: "/about"},
							layout.NavLink{Label: "Contact", Href: "/contact"},
						},
					},
				},
			},
		},
		{
			name: "current path marks active link",
			props: layout.NavbarProps{
				Brand:       layout.NavbarBrand{Label: "MyApp", Href: "/"},
				CurrentPath: "/about",
				Sections: []layout.NavbarSection{
					{
						Items: []layout.NavbarItem{
							layout.NavLink{Label: "Home", Href: "/"},
							layout.NavLink{Label: "About", Href: "/about"},
						},
					},
				},
			},
		},
		{
			name: "with separator and text",
			props: layout.NavbarProps{
				Brand: layout.NavbarBrand{Label: "MyApp", Href: "/"},
				Sections: []layout.NavbarSection{
					{
						Items: []layout.NavbarItem{
							layout.NavLink{Label: "Home", Href: "/"},
							layout.NavSeparator{},
							layout.NavText{Text: "Welcome"},
						},
					},
				},
			},
		},
		{
			name: "brand with logo path",
			props: layout.NavbarProps{
				Brand: layout.NavbarBrand{
					Label:    "MyApp",
					Href:     "/",
					LogoPath: "/logo.png",
				},
				Sections: []layout.NavbarSection{
					{
						Items: []layout.NavbarItem{
							layout.NavLink{Label: "Home", Href: "/"},
						},
					},
				},
			},
		},
		{
			name: "with prefix match active",
			props: layout.NavbarProps{
				Brand:       layout.NavbarBrand{Label: "MyApp", Href: "/"},
				CurrentPath: "/blog/post-1",
				Sections: []layout.NavbarSection{
					{
						Items: []layout.NavbarItem{
							layout.NavLink{Label: "Blog", Href: "/blog", MatchPrefix: true},
						},
					},
				},
			},
		},
		{
			name: "with custom id",
			props: layout.NavbarProps{
				ID:    "main-nav",
				Brand: layout.NavbarBrand{Label: "MyApp", Href: "/"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := testutil.RenderNode(t, layout.Navbar(tc.props))
			want := testutil.LoadGolden(t)
			testutil.AssertEqualHTML(t, want, got)
		})
	}
}

func TestNavbar_DefaultBrand(t *testing.T) {
	got := testutil.RenderNode(t, layout.Navbar(layout.NavbarProps{}))
	// default brand label is "Starter" and href is "/"
	if !containsStr(got, "Starter") {
		t.Errorf("expected default brand label 'Starter' in output, got:\n%s", got)
	}
	if !containsStr(got, `href="/"`) {
		t.Errorf("expected default brand href '/' in output, got:\n%s", got)
	}
}

func TestNavbar_NavLink_active(t *testing.T) {
	got := testutil.RenderNode(t, layout.Navbar(layout.NavbarProps{
		Brand:       layout.NavbarBrand{Label: "App", Href: "/"},
		CurrentPath: "/dashboard",
		Sections: []layout.NavbarSection{
			{Items: []layout.NavbarItem{layout.NavLink{Label: "Dashboard", Href: "/dashboard"}}},
		},
	}))
	if !containsStr(got, `aria-current="page"`) {
		t.Errorf("expected aria-current=page for active link, got:\n%s", got)
	}
}

func TestNavbar_NavLink_inactive(t *testing.T) {
	got := testutil.RenderNode(t, layout.Navbar(layout.NavbarProps{
		Brand:       layout.NavbarBrand{Label: "App", Href: "/"},
		CurrentPath: "/other",
		Sections: []layout.NavbarSection{
			{Items: []layout.NavbarItem{layout.NavLink{Label: "Dashboard", Href: "/dashboard"}}},
		},
	}))
	if containsStr(got, `aria-current="page"`) {
		t.Errorf("expected no aria-current for inactive link, got:\n%s", got)
	}
}

func TestNavbar_NavLink_visibilityAll(t *testing.T) {
	for _, auth := range []bool{true, false} {
		got := testutil.RenderNode(t, layout.Navbar(layout.NavbarProps{
			Brand:           layout.NavbarBrand{Label: "App", Href: "/"},
			IsAuthenticated: auth,
			Sections: []layout.NavbarSection{
				{Items: []layout.NavbarItem{layout.NavLink{Visibility: layout.VisibilityAll, Label: "Public", Href: "/public"}}},
			},
		}))
		if !containsStr(got, "Public") {
			t.Errorf("VisibilityAll with auth=%v: expected 'Public' in output, got:\n%s", auth, got)
		}
	}
}

func TestNavbar_NavLink_visibilityUser_authenticated(t *testing.T) {
	got := testutil.RenderNode(t, layout.Navbar(layout.NavbarProps{
		Brand:           layout.NavbarBrand{Label: "App", Href: "/"},
		IsAuthenticated: true,
		Sections: []layout.NavbarSection{
			{Items: []layout.NavbarItem{layout.NavLink{Visibility: layout.VisibilityUser, Label: "Profile", Href: "/profile"}}},
		},
	}))
	if !containsStr(got, "Profile") {
		t.Errorf("VisibilityUser authenticated: expected 'Profile' in output, got:\n%s", got)
	}
}

func TestNavbar_NavLink_visibilityUser_guest(t *testing.T) {
	got := testutil.RenderNode(t, layout.Navbar(layout.NavbarProps{
		Brand:           layout.NavbarBrand{Label: "App", Href: "/"},
		IsAuthenticated: false,
		Sections: []layout.NavbarSection{
			{Items: []layout.NavbarItem{layout.NavLink{Visibility: layout.VisibilityUser, Label: "Profile", Href: "/profile"}}},
		},
	}))
	if containsStr(got, "Profile") {
		t.Errorf("VisibilityUser guest: expected 'Profile' to be hidden, got:\n%s", got)
	}
}

func TestNavbar_NavLink_visibilityGuest_guest(t *testing.T) {
	got := testutil.RenderNode(t, layout.Navbar(layout.NavbarProps{
		Brand:           layout.NavbarBrand{Label: "App", Href: "/"},
		IsAuthenticated: false,
		Sections: []layout.NavbarSection{
			{Items: []layout.NavbarItem{layout.NavLink{Visibility: layout.VisibilityGuest, Label: "Login", Href: "/login"}}},
		},
	}))
	if !containsStr(got, "Login") {
		t.Errorf("VisibilityGuest guest: expected 'Login' in output, got:\n%s", got)
	}
}

func TestNavbar_NavLink_visibilityGuest_authenticated(t *testing.T) {
	got := testutil.RenderNode(t, layout.Navbar(layout.NavbarProps{
		Brand:           layout.NavbarBrand{Label: "App", Href: "/"},
		IsAuthenticated: true,
		Sections: []layout.NavbarSection{
			{Items: []layout.NavbarItem{layout.NavLink{Visibility: layout.VisibilityGuest, Label: "Login", Href: "/login"}}},
		},
	}))
	if containsStr(got, "Login") {
		t.Errorf("VisibilityGuest authenticated: expected 'Login' to be hidden, got:\n%s", got)
	}
}

func TestNavbar_NavGroup(t *testing.T) {
	got := testutil.RenderNode(t, layout.Navbar(layout.NavbarProps{
		Brand: layout.NavbarBrand{Label: "App", Href: "/"},
		Sections: []layout.NavbarSection{
			{Items: []layout.NavbarItem{
				layout.NavGroup{
					Label: "Products",
					Items: []layout.NavbarItem{
						layout.NavLink{Label: "Widget A", Href: "/products/a"},
					},
				},
			}},
		},
	}))
	// NavGroup renders as <details>/<summary>
	if !containsStr(got, "<details") {
		t.Errorf("expected <details> element for NavGroup, got:\n%s", got)
	}
	if !containsStr(got, "<summary") {
		t.Errorf("expected <summary> element for NavGroup, got:\n%s", got)
	}
	if !containsStr(got, "Products") {
		t.Errorf("expected 'Products' label in NavGroup output, got:\n%s", got)
	}
}

func TestNavbar_NavForm(t *testing.T) {
	got := testutil.RenderNode(t, layout.Navbar(layout.NavbarProps{
		Brand: layout.NavbarBrand{Label: "App", Href: "/"},
		Sections: []layout.NavbarSection{
			{Items: []layout.NavbarItem{
				layout.NavForm{
					Label:  "Sign out",
					Action: "/signout",
					Method: "post",
				},
			}},
		},
	}))
	if !containsStr(got, `action="/signout"`) {
		t.Errorf("expected action=/signout in form output, got:\n%s", got)
	}
	if !containsStr(got, "<form") {
		t.Errorf("expected <form> element in NavForm output, got:\n%s", got)
	}
}

func TestNavbar_NavSeparator(t *testing.T) {
	// NavSeparator renders as a role=separator div in mobile view
	// In desktop at depth=0 it returns nil; test via mobile presence in output
	got := testutil.RenderNode(t, layout.Navbar(layout.NavbarProps{
		Brand: layout.NavbarBrand{Label: "App", Href: "/"},
		Sections: []layout.NavbarSection{
			{Items: []layout.NavbarItem{
				layout.NavLink{Label: "Home", Href: "/"},
				layout.NavSeparator{},
				layout.NavLink{Label: "About", Href: "/about"},
			}},
		},
	}))
	// Separator shows in mobile drawer (md:hidden section)
	if !containsStr(got, `role="separator"`) {
		t.Errorf("expected role=separator in navbar output (mobile), got:\n%s", got)
	}
}

func TestNavbar_brand_withLogoPath(t *testing.T) {
	got := testutil.RenderNode(t, layout.Navbar(layout.NavbarProps{
		Brand: layout.NavbarBrand{
			Label:    "MyBrand",
			Href:     "/",
			LogoPath: "/static/logo.svg",
		},
	}))
	if !containsStr(got, `src="/static/logo.svg"`) {
		t.Errorf("expected img src='/static/logo.svg', got:\n%s", got)
	}
	if !containsStr(got, "<img") {
		t.Errorf("expected <img> tag in brand area, got:\n%s", got)
	}
}

func TestNavbar_alignEnd(t *testing.T) {
	got := testutil.RenderNode(t, layout.Navbar(layout.NavbarProps{
		Brand: layout.NavbarBrand{Label: "App", Href: "/"},
		Sections: []layout.NavbarSection{
			{Align: layout.AlignEnd, Items: []layout.NavbarItem{
				layout.NavLink{Label: "Login", Href: "/login"},
			}},
		},
	}))
	// Should render without panicking and contain the link label
	if !containsStr(got, "Login") {
		t.Errorf("expected 'Login' in end-aligned section output, got:\n%s", got)
	}
}

// ---- NavNode tests ----

func TestNavNode_RenderFn_called(t *testing.T) {
	called := false
	node := layout.NavNode{
		RenderFn: func(ctx layout.NavbarContext, isDesktop bool) g.Node {
			called = true
			return h.Span(g.Text("from-renderfn"))
		},
	}
	got := renderNodeLayout(t, layout.Navbar(layout.NavbarProps{
		Brand: layout.NavbarBrand{Label: "App", Href: "/"},
		Sections: []layout.NavbarSection{
			{Items: []layout.NavbarItem{node}},
		},
	}))

	if !called {
		t.Error("expected RenderFn to be called, but it was not")
	}
	if !containsStr(got, "from-renderfn") {
		t.Errorf("expected 'from-renderfn' in output, got:\n%s", got)
	}
}

func TestNavNode_RenderFn_nil_usesDesktopNode(t *testing.T) {
	node := layout.NavNode{
		Desktop: h.Span(g.Text("desktop-node")),
		Mobile:  h.Span(g.Text("mobile-node")),
	}
	got := renderNodeLayout(t, layout.Navbar(layout.NavbarProps{
		Brand: layout.NavbarBrand{Label: "App", Href: "/"},
		Sections: []layout.NavbarSection{
			{Items: []layout.NavbarItem{node}},
		},
	}))

	// Both desktop and mobile nodes should appear (navbar renders both regions)
	if !containsStr(got, "desktop-node") {
		t.Errorf("expected 'desktop-node' in output (RenderFn=nil fallback), got:\n%s", got)
	}
	if !containsStr(got, "mobile-node") {
		t.Errorf("expected 'mobile-node' in output (RenderFn=nil fallback), got:\n%s", got)
	}
}

// ---- RenderNavText tests ----

func TestRenderNavText_desktop_nonNil(t *testing.T) {
	ctx := layout.NavbarContext{CurrentPath: "/", IsAuthenticated: false}
	node := layout.RenderNavText("Hello", "", ctx, true, 0)
	if node == nil {
		t.Fatal("expected non-nil node from RenderNavText for desktop")
	}
	got := renderNodeLayout(t, node)
	if !containsStr(got, "Hello") {
		t.Errorf("expected 'Hello' in RenderNavText desktop output, got:\n%s", got)
	}
}

func TestRenderNavText_mobile_nonNil(t *testing.T) {
	ctx := layout.NavbarContext{CurrentPath: "/", IsAuthenticated: false}
	node := layout.RenderNavText("World", "", ctx, false, 0)
	if node == nil {
		t.Fatal("expected non-nil node from RenderNavText for mobile")
	}
	got := renderNodeLayout(t, node)
	if !containsStr(got, "World") {
		t.Errorf("expected 'World' in RenderNavText mobile output, got:\n%s", got)
	}
}

func TestRenderNavText_emptyTextReturnsNil(t *testing.T) {
	ctx := layout.NavbarContext{}
	node := layout.RenderNavText("", "", ctx, true, 0)
	if node != nil {
		t.Errorf("expected nil node for empty text, got non-nil")
	}
}

// ---- RenderNavForm tests ----

func TestRenderNavForm_desktop_nonNil(t *testing.T) {
	ctx := layout.NavbarContext{}
	form := layout.NavForm{
		Label:  "Sign out",
		Action: "/signout",
		Method: "post",
	}
	node := layout.RenderNavForm(form, ctx, true, 0)
	if node == nil {
		t.Fatal("expected non-nil node from RenderNavForm for desktop")
	}
	got := renderNodeLayout(t, node)
	if !containsStr(got, `action="/signout"`) {
		t.Errorf("expected action=/signout in RenderNavForm desktop output, got:\n%s", got)
	}
}

func TestRenderNavForm_mobile_nonNil(t *testing.T) {
	ctx := layout.NavbarContext{}
	form := layout.NavForm{
		Label:  "Sign out",
		Action: "/signout",
		Method: "post",
	}
	node := layout.RenderNavForm(form, ctx, false, 0)
	if node == nil {
		t.Fatal("expected non-nil node from RenderNavForm for mobile")
	}
	got := renderNodeLayout(t, node)
	if !containsStr(got, `action="/signout"`) {
		t.Errorf("expected action=/signout in RenderNavForm mobile output, got:\n%s", got)
	}
}

func TestRenderNavForm_emptyActionReturnsNil(t *testing.T) {
	ctx := layout.NavbarContext{}
	form := layout.NavForm{Label: "Sign out"} // no Action
	node := layout.RenderNavForm(form, ctx, true, 0)
	if node != nil {
		t.Errorf("expected nil node for form with empty action, got non-nil")
	}
}
