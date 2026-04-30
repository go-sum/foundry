package config

import (
	"cmp"

	"github.com/go-sum/foundry/pkg/componentry/compound"
	"github.com/go-sum/foundry/pkg/componentry/icons"
	"github.com/go-sum/foundry/pkg/componentry/icons/render"
	"github.com/go-sum/foundry/pkg/componentry/interactive/theme"
	"github.com/go-sum/foundry/pkg/componentry/ui/core"
	"github.com/go-sum/foundry/pkg/web/authn"
	"github.com/go-sum/foundry/pkg/web/router"
	viewstate "github.com/go-sum/foundry/pkg/web/viewstate"
)

const maxLen = 12

// DefaultNav returns a viewstate.RequestOption that builds the nav on each request.
// Route URL reversal is deferred until request time so this option is safe to
// construct before routes are registered. The CSRF token for the signout form
// is injected from the per-request viewstate.Request.
func DefaultNav(rt *router.Router, authRoute string) viewstate.RequestOption {
	return func(r *viewstate.Request) {
		name := r.Auth.DisplayName
		if runes := []rune(name); len(runes) > maxLen {
			name = string(runes[:maxLen])
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
								{Label: "Sign up", Href: rt.MustReverse(authRoute, nil), Visibility: compound.NavVisibilityGuest},
								{
									Label:  "Sign out",
									Action: rt.MustReverse(authn.RouteSignout, nil),
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
