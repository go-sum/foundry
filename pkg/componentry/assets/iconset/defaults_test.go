package iconset

import (
	"testing"

	"github.com/go-sum/componentry/icons"
)

func TestDefaultSprites(t *testing.T) {
	for _, key := range []string{"lucide", "theme"} {
		if _, ok := Default.Sprites[key]; !ok {
			t.Errorf("Default.Sprites missing key %q", key)
		}
	}
}

func TestDefaultIconsAllKeysPresent(t *testing.T) {
	keys := []icons.Key{
		icons.ChevronDown,
		icons.ChevronLeft,
		icons.ChevronRight,
		icons.ChevronsUp,
		icons.Close,
		icons.ThemeLight,
		icons.ThemeDark,
		icons.ThemeSystem,
	}
	for _, key := range keys {
		if _, ok := Default.Icons[key]; !ok {
			t.Errorf("Default.Icons missing key %q", key)
		}
	}
}

func TestInstallTo(t *testing.T) {
	r := icons.NewRegistry()
	InstallTo(r)

	tests := []struct {
		key  icons.Key
		want icons.Ref
	}{
		{icons.ChevronDown, icons.Ref{Sprite: "lucide", ID: "chevron-down"}},
		{icons.ChevronLeft, icons.Ref{Sprite: "lucide", ID: "chevron-left"}},
		{icons.ChevronRight, icons.Ref{Sprite: "lucide", ID: "chevron-right"}},
		{icons.ChevronsUp, icons.Ref{Sprite: "lucide", ID: "chevrons-up"}},
		{icons.Close, icons.Ref{Sprite: "lucide", ID: "x"}},
		{icons.ThemeLight, icons.Ref{Sprite: "theme", ID: "sun"}},
		{icons.ThemeDark, icons.Ref{Sprite: "theme", ID: "moon"}},
		{icons.ThemeSystem, icons.Ref{Sprite: "theme", ID: "monitor"}},
	}

	for _, tt := range tests {
		got, ok := r.Resolve(tt.key)
		if !ok {
			t.Errorf("InstallTo: key %q not registered", tt.key)
			continue
		}
		if got != tt.want {
			t.Errorf("InstallTo: key %q = %+v, want %+v", tt.key, got, tt.want)
		}
	}
}

