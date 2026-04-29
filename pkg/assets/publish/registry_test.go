package publish

import (
	"testing"
)

func TestRegistry_RegisterSprite(t *testing.T) {
	r := NewRegistry()
	r.RegisterSprite("icons", "img/svg/icons.svg")
	got := r.SpritePath("icons")
	if got != "/public/img/svg/icons.svg" {
		t.Errorf("SpritePath = %q, want %q", got, "/public/img/svg/icons.svg")
	}
}

func TestRegistry_SpritePath_miss(t *testing.T) {
	r := NewRegistry()
	got := r.SpritePath("nonexistent")
	if got != "" {
		t.Errorf("SpritePath miss = %q, want empty string", got)
	}
}

func TestRegistry_SetPathFunc(t *testing.T) {
	r := NewRegistry()
	r.RegisterSprite("icons", "img/svg/icons.svg")
	r.SetPathFunc(func(rel string) string {
		return "/custom/" + rel
	})
	got := r.SpritePath("icons")
	if got != "/custom/img/svg/icons.svg" {
		t.Errorf("SpritePath with custom func = %q, want %q", got, "/custom/img/svg/icons.svg")
	}
}

func TestRegistry_RegisterSprites(t *testing.T) {
	r := NewRegistry()
	r.RegisterSprites(map[string]string{
		"icons":  "img/svg/icons.svg",
		"arrows": "img/svg/arrows.svg",
	})
	if got := r.SpritePath("icons"); got != "/public/img/svg/icons.svg" {
		t.Errorf("icons = %q, want %q", got, "/public/img/svg/icons.svg")
	}
	if got := r.SpritePath("arrows"); got != "/public/img/svg/arrows.svg" {
		t.Errorf("arrows = %q, want %q", got, "/public/img/svg/arrows.svg")
	}
}

