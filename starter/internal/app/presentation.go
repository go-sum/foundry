package app

import (
	"cmp"

	"github.com/go-sum/foundry/pkg/componentry/compound"
	"github.com/go-sum/foundry/pkg/componentry/icons"
	"github.com/go-sum/foundry/pkg/componentry/icons/render"
	"github.com/go-sum/foundry/pkg/componentry/interactive/theme"
	"github.com/go-sum/foundry/pkg/componentry/ui/core"
	"github.com/go-sum/foundry/pkg/web"
	"github.com/go-sum/foundry/pkg/web/authn"
	"github.com/go-sum/foundry/pkg/web/router"
	viewstate "github.com/go-sum/foundry/pkg/web/viewstate"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

const navAccountLabelMaxLen = 12

// primaryNav builds the starter navigation on each request.
func primaryNav(rt *router.Router, authRoute string) viewstate.RequestOption {
	return func(r *viewstate.Request) {
		name := r.Auth.DisplayName
		if runes := []rune(name); len(runes) > navAccountLabelMaxLen {
			name = string(runes[:navAccountLabelMaxLen])
		}
		accountLabel := cmp.Or(name, "Account")
		r.NavConfig = compound.NavConfig{
			Brand: compound.NavBrand{
				Label: "Starter",
				Href:  rt.MustReverse("home.show", nil),
			},
			Sections: []compound.NavSection{
				{
					Items: []compound.NavItem{
						{Label: "Home", Href: rt.MustReverse("home.show", nil)},
						{
							Label: "Packages",
							Items: []compound.NavItem{
								{
									Label: "Showcase",
									Items: []compound.NavItem{
										{Label: "Components", Href: rt.MustReverse("demos.showcase", nil)},
										{Label: "Database", Href: rt.MustReverse("db.index", nil)},
										{Label: "Key-Value", Href: rt.MustReverse("kv.index", nil)},
										{Label: "Queues", Href: rt.MustReverse("queue.index", nil)},
									},
								},
								{
									Label: "Site",
									Items: []compound.NavItem{
										{Label: "Robots", Href: rt.MustReverse("meta.robots", nil)},
										{Label: "Sitemap", Href: rt.MustReverse("meta.sitemap", nil)},
									},
								},
							},
						},
					},
				},
				{
					Align: compound.NavAlignEnd,
					Items: []compound.NavItem{
						{Label: "Documentation", Href: rt.MustReverse("docs.index", nil), Visibility: compound.NavVisibilityUser},
						{Label: "Contact", Href: rt.MustReverse("contact.form", nil)},
						{
							Label: accountLabel,
							Items: []compound.NavItem{
								{Label: "Sign in", Href: rt.MustReverse(authRoute, nil), Visibility: compound.NavVisibilityGuest},
								{Label: "Sign up", Href: rt.MustReverse(authn.DefaultRouteConfig().Signup.Name, nil), Visibility: compound.NavVisibilityGuest},
								{
									Label:  "Sign out",
									Action: rt.MustReverse(authn.DefaultRouteConfig().Signout.Name, nil),
									Method: "POST",
									HiddenFields: []compound.NavHiddenField{
										{Name: r.CSRFFieldName, Value: r.CSRFToken},
									},
									Visibility: compound.NavVisibilityUser,
								},
							},
						},
						{Slot: compound.SlotThemeToggle},
					},
				},
			},
			Slots: compound.NavSlots{
				compound.SlotThemeToggle: compound.ControlSlot("Theme", theme.ThemeSelector(theme.ThemeSelectorProps{
					LightIcon:  core.Icon(render.PropsForRegistry(r.Icons, icons.ThemeLight, core.IconProps{})),
					DarkIcon:   core.Icon(render.PropsForRegistry(r.Icons, icons.ThemeDark, core.IconProps{})),
					SystemIcon: core.Icon(render.PropsForRegistry(r.Icons, icons.ThemeSystem, core.IconProps{})),
				})),
			},
		}
	}
}

func pageRenderer(opts []viewstate.RequestOption) func(c *web.Context, title string, content g.Node) (web.Response, error) {
	return func(c *web.Context, title string, content g.Node) (web.Response, error) {
		vr := viewstate.NewRequest(c, opts...)
		return viewstate.Render(vr, vr.Page(title, content), nil)
	}
}

func centeredAuthPageRenderer(opts []viewstate.RequestOption) func(c *web.Context, title string, content g.Node) (web.Response, error) {
	return func(c *web.Context, title string, content g.Node) (web.Response, error) {
		vr := viewstate.NewRequest(c, opts...)
		centered := h.Div(
			h.Class("flex min-h-[calc(100vh-4rem)] items-center justify-center px-4"),
			h.Div(h.Class("w-full max-w-sm"), content),
		)
		return viewstate.Render(vr, vr.Page(title, centered), content)
	}
}
