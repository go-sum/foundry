package iconset

import "github.com/go-sum/componentry/icons"

// Catalog maps semantic icon keys to their sprite and symbol IDs.
type Catalog struct {
	Sprites map[string]string       // sprite key → file path
	Icons   map[icons.Key]icons.Ref // semantic key → Ref{Sprite, ID}
}

// Default is the built-in icon catalog.
// It expects two sprite files to be served at the configured paths:
//   - "lucide" sprite: img/svg/lucide-icons.svg
//   - "theme" sprite:  img/svg/theme-icons.svg
var Default = Catalog{
	Sprites: map[string]string{
		"lucide": "img/svg/lucide-icons.svg",
		"theme":  "img/svg/theme-icons.svg",
	},
	Icons: map[icons.Key]icons.Ref{
		icons.ChevronDown:  {Sprite: "lucide", ID: "chevron-down"},
		icons.ChevronLeft:  {Sprite: "lucide", ID: "chevron-left"},
		icons.ChevronRight: {Sprite: "lucide", ID: "chevron-right"},
		icons.ChevronsUp:   {Sprite: "lucide", ID: "chevrons-up"},
		icons.Close:        {Sprite: "lucide", ID: "x"},
		icons.ThemeLight:   {Sprite: "theme", ID: "sun"},
		icons.ThemeDark:    {Sprite: "theme", ID: "moon"},
		icons.ThemeSystem:  {Sprite: "theme", ID: "monitor"},
	},
}

// Install registers Default onto the package-level icons.Default registry.
// Call this once at application startup before rendering any components.
func Install() {
	InstallTo(icons.Default)
}

// InstallTo registers Default onto the specified registry.
// Use when managing multiple isolated registries.
func InstallTo(r *icons.Registry) {
	r.RegisterSet(Default.Icons)
}
