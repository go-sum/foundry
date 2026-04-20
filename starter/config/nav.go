package config

import (
	"github.com/go-sum/componentry/compound"
	"github.com/go-sum/componentry/interactive/theme"
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
								Label: "Componentry",
								Items: []compound.NavItem{
									{Label: "Components", Href: "/componentry/_components"},
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
					{Slot: compound.SlotThemeToggle},
				},
			},
		},
		Slots: compound.NavSlots{
			compound.SlotThemeToggle: compound.ControlSlot("Theme", theme.ThemeSelector(theme.ThemeSelectorProps{})),
		},
	}
}
