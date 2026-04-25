package config

import (
	"github.com/go-sum/componentry/compound"
	"github.com/go-sum/componentry/icons"
	"github.com/go-sum/componentry/icons/render"
	"github.com/go-sum/componentry/interactive/theme"
	"github.com/go-sum/componentry/ui/core"
	"github.com/go-sum/web/router"
)

// DefaultNav returns the default navigation configuration for the Starter application.
func DefaultNav(rt *router.Router) compound.NavConfig {
	return compound.NavConfig{
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
					{Label: "Documentation", Href: rt.MustReverse("docs.index", nil)},
					{Label: "Contact", Href: rt.MustReverse("contact.form", nil)},
					{Slot: compound.SlotThemeToggle},
				},
			},
		},
		Slots: compound.NavSlots{
			compound.SlotThemeToggle: compound.ControlSlot("Theme", theme.ThemeSelector(theme.ThemeSelectorProps{
				LightIcon:  core.Icon(render.PropsForRegistry(nil, icons.ThemeLight, core.IconProps{})),
				DarkIcon:   core.Icon(render.PropsForRegistry(nil, icons.ThemeDark, core.IconProps{})),
				SystemIcon: core.Icon(render.PropsForRegistry(nil, icons.ThemeSystem, core.IconProps{})),
			})),
		},
	}
}
