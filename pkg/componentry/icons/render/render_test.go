package render_test

import (
	"testing"

	"github.com/go-sum/foundry/pkg/componentry/icons"
	iconrender "github.com/go-sum/foundry/pkg/componentry/icons/render"
	"github.com/go-sum/foundry/pkg/componentry/ui/core"
)

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

func TestPropsForRegistry_nilRegistry(t *testing.T) {
	const testKey icons.Key = "any-key"
	base := core.IconProps{Src: "/base.svg", ID: "base-id", Label: "Base", Size: "size-4"}

	got := iconrender.PropsForRegistry(nil, testKey, base)

	// nil registry: base returned unchanged
	if got.Src != base.Src {
		t.Errorf("PropsForRegistry nil registry: Src = %q, want %q", got.Src, base.Src)
	}
	if got.ID != base.ID {
		t.Errorf("PropsForRegistry nil registry: ID = %q, want %q", got.ID, base.ID)
	}
	if got.Label != base.Label {
		t.Errorf("PropsForRegistry nil registry: Label = %q, want %q", got.Label, base.Label)
	}
	if got.Size != base.Size {
		t.Errorf("PropsForRegistry nil registry: Size = %q, want %q", got.Size, base.Size)
	}
}
