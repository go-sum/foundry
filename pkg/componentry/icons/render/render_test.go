package render_test

import (
	"testing"

	"github.com/go-sum/componentry/icons"
	iconrender "github.com/go-sum/componentry/icons/render"
	"github.com/go-sum/componentry/ui/core"
)

func TestPropsFor_found(t *testing.T) {
	// Register on Default so PropsFor can find it.
	const testKey icons.Key = "render-test-found"
	icons.Register(testKey, icons.Ref{Sprite: "/sprite.svg", ID: "found-icon"})

	base := core.IconProps{Label: "Test Icon", Size: "size-6"}
	got := iconrender.PropsFor(testKey, base)

	if got.Src != "/sprite.svg" {
		t.Errorf("PropsFor found: Src = %q, want %q", got.Src, "/sprite.svg")
	}
	if got.ID != "found-icon" {
		t.Errorf("PropsFor found: ID = %q, want %q", got.ID, "found-icon")
	}
	// base fields should be preserved
	if got.Label != base.Label {
		t.Errorf("PropsFor found: Label = %q, want %q", got.Label, base.Label)
	}
	if got.Size != base.Size {
		t.Errorf("PropsFor found: Size = %q, want %q", got.Size, base.Size)
	}
}

func TestPropsFor_notFound(t *testing.T) {
	const testKey icons.Key = "render-test-notfound-xyz"
	base := core.IconProps{Label: "Fallback", Size: "size-8"}

	got := iconrender.PropsFor(testKey, base)

	// Unregistered: base returned unchanged
	if got.Src != base.Src {
		t.Errorf("PropsFor notFound: Src = %q, want %q", got.Src, base.Src)
	}
	if got.ID != base.ID {
		t.Errorf("PropsFor notFound: ID = %q, want %q", got.ID, base.ID)
	}
	if got.Label != base.Label {
		t.Errorf("PropsFor notFound: Label = %q, want %q", got.Label, base.Label)
	}
	if got.Size != base.Size {
		t.Errorf("PropsFor notFound: Size = %q, want %q", got.Size, base.Size)
	}
}

func TestPropsForRegistry_explicit(t *testing.T) {
	// Isolated registry — not shared with Default
	r := icons.NewRegistry()
	const testKey icons.Key = "explicit-registry-key"
	r.Register(testKey, icons.Ref{Sprite: "/explicit.svg", ID: "explicit-id"})

	base := core.IconProps{Size: "size-4"}
	got := iconrender.PropsForRegistry(r, testKey, base)

	if got.Src != "/explicit.svg" {
		t.Errorf("PropsForRegistry: Src = %q, want %q", got.Src, "/explicit.svg")
	}
	if got.ID != "explicit-id" {
		t.Errorf("PropsForRegistry: ID = %q, want %q", got.ID, "explicit-id")
	}
	if got.Size != "size-4" {
		t.Errorf("PropsForRegistry: Size = %q, want %q", got.Size, "size-4")
	}
}

func TestPropsForRegistry_notFound(t *testing.T) {
	r := icons.NewRegistry()
	const testKey icons.Key = "registry-miss"
	base := core.IconProps{Src: "/fallback.svg", ID: "fallback"}

	got := iconrender.PropsForRegistry(r, testKey, base)

	if got.Src != base.Src {
		t.Errorf("PropsForRegistry notFound: Src = %q, want %q", got.Src, base.Src)
	}
	if got.ID != base.ID {
		t.Errorf("PropsForRegistry notFound: ID = %q, want %q", got.ID, base.ID)
	}
}
