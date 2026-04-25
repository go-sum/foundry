package config

import (
	"github.com/go-sum/componentry/compound"
	"github.com/go-sum/componentry/icons"
	"github.com/go-sum/componentry/icons/render"
	"github.com/go-sum/componentry/interactive/theme"
	"github.com/go-sum/componentry/ui/core"
)

// DefaultNav returns the default navigation configuration for the Starter application.
func DefaultNav() compound.NavConfig {
	return compound.NavConfig{
		Brand: compound.NavBrand{
			Label: "Starter",
			Href:  "/",
		},
		Sections: []compound.NavSection{
			{
				Items: []compound.NavItem{
					{Label: "Home", Href: "/"},
					{
						Label: "Packages",
						Items: []compound.NavItem{
							{
								Label: "Showcase",
								Items: []compound.NavItem{
									{Label: "Components", Href: "/showcase/componentry/components"},
									{Label: "Database", Href: "/showcase/db"},
									{Label: "Key-Value", Href: "/showcase/kv"},
									{Label: "Queues", Href: "/showcase/queues"},
								},
							},
							{
								Label: "Site",
								Items: []compound.NavItem{
									{Label: "Robots", Href: "/robots.txt"},
									{Label: "Sitemap", Href: "/sitemap.xml"},
								},
							},
						},
					},
				},
			},
			{
				Align: compound.NavAlignEnd,
				Items: []compound.NavItem{
					{Label: "Documentation", Href: "/docs"},
					{Label: "Contact", Href: "/contact"},
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
