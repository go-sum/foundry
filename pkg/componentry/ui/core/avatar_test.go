package core_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/componentry/ui/core"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestAvatar_Image(t *testing.T) {
	got := testutil.RenderNode(t, core.Avatar.Image("/avatar.jpg", "Jane Doe"))
	if !strings.Contains(got, `src="/avatar.jpg"`) {
		t.Errorf("Avatar.Image: expected src=/avatar.jpg, got:\n%s", got)
	}
	if !strings.Contains(got, `alt="Jane Doe"`) {
		t.Errorf("Avatar.Image: expected alt=Jane Doe, got:\n%s", got)
	}
	// outer span carries the root class
	if !strings.Contains(got, "relative flex h-10 w-10 shrink-0 overflow-hidden rounded-full") {
		t.Errorf("Avatar.Image: expected root class in output, got:\n%s", got)
	}
	// image carries the image class
	if !strings.Contains(got, "aspect-square h-full w-full object-cover") {
		t.Errorf("Avatar.Image: expected image class in output, got:\n%s", got)
	}
}

func TestAvatar_Image_extra(t *testing.T) {
	got := testutil.RenderNode(t, core.Avatar.Image("/avatar.jpg", "Jane", g.Attr("data-test", "avatar")))
	if !strings.Contains(got, `data-test="avatar"`) {
		t.Errorf("Avatar.Image extra: expected data-test attribute, got:\n%s", got)
	}
}

func TestAvatar_Fallback(t *testing.T) {
	got := testutil.RenderNode(t, core.Avatar.Fallback(g.Text("JD")))
	// outer span
	if !strings.Contains(got, "relative flex h-10 w-10 shrink-0 overflow-hidden rounded-full") {
		t.Errorf("Avatar.Fallback: expected root class, got:\n%s", got)
	}
	// inner span with bg-muted
	if !strings.Contains(got, "bg-muted") {
		t.Errorf("Avatar.Fallback: expected bg-muted class in inner span, got:\n%s", got)
	}
	if !strings.Contains(got, "JD") {
		t.Errorf("Avatar.Fallback: expected children 'JD' in output, got:\n%s", got)
	}
}
