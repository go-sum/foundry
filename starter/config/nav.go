package config

import "github.com/go-sum/componentry/compound"

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
				},
			},
			{
				Align: compound.NavAlignEnd,
				Items: []compound.NavItem{
					{Label: "Sign In", Href: "/signin", Visibility: compound.NavVisibilityGuest},
					{Label: "Sign Up", Href: "/signup", Visibility: compound.NavVisibilityGuest},
					{Slot: "theme_toggle"},
				},
			},
		},
	}
}
