package core_test

import (
	"strings"
	"testing"

	"github.com/go-sum/componentry/ui/core"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestIcon_decorative(t *testing.T) {
	got := testutil.RenderNode(t, core.Icon(core.IconProps{
		Src: "/icons.svg",
		ID:  "star",
	}))
	if !strings.Contains(got, `aria-hidden="true"`) {
		t.Errorf("decorative icon: expected aria-hidden=true, got:\n%s", got)
	}
	if strings.Contains(got, `role="img"`) {
		t.Errorf("decorative icon: expected no role=img, got:\n%s", got)
	}
	if strings.Contains(got, `aria-label`) {
		t.Errorf("decorative icon: expected no aria-label, got:\n%s", got)
	}
}

func TestIcon_labelled(t *testing.T) {
	got := testutil.RenderNode(t, core.Icon(core.IconProps{
		Src:   "/icons.svg",
		ID:    "star",
		Label: "Favourite",
	}))
	if !strings.Contains(got, `role="img"`) {
		t.Errorf("labelled icon: expected role=img, got:\n%s", got)
	}
	if !strings.Contains(got, `aria-label="Favourite"`) {
		t.Errorf("labelled icon: expected aria-label=Favourite, got:\n%s", got)
	}
	if strings.Contains(got, `aria-hidden`) {
		t.Errorf("labelled icon: expected no aria-hidden, got:\n%s", got)
	}
}

func TestIcon_defaultSize(t *testing.T) {
	got := testutil.RenderNode(t, core.Icon(core.IconProps{
		Src: "/icons.svg",
		ID:  "star",
	}))
	if !strings.Contains(got, "size-4") {
		t.Errorf("default size: expected class 'size-4', got:\n%s", got)
	}
}

func TestIcon_customSize(t *testing.T) {
	got := testutil.RenderNode(t, core.Icon(core.IconProps{
		Src:  "/icons.svg",
		ID:   "star",
		Size: "size-8",
	}))
	if !strings.Contains(got, "size-8") {
		t.Errorf("custom size: expected class 'size-8', got:\n%s", got)
	}
	if strings.Contains(got, "size-4") {
		t.Errorf("custom size: expected no class 'size-4', got:\n%s", got)
	}
}

func TestIcon_hrefWithSprite(t *testing.T) {
	got := testutil.RenderNode(t, core.Icon(core.IconProps{
		Src: "/icons.svg",
		ID:  "star",
	}))
	if !strings.Contains(got, `href="/icons.svg#star"`) {
		t.Errorf("icon href: expected href=/icons.svg#star, got:\n%s", got)
	}
}

func TestIcon_hrefWithoutSprite(t *testing.T) {
	got := testutil.RenderNode(t, core.Icon(core.IconProps{
		ID: "star",
	}))
	if !strings.Contains(got, `href="#star"`) {
		t.Errorf("icon href no sprite: expected href=#star, got:\n%s", got)
	}
}
