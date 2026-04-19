package core_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/componentry/ui/core"
	testutil "github.com/go-sum/componentry/testutil"
)

func TestSkeleton(t *testing.T) {
	got := testutil.RenderNode(t, core.Skeleton())
	if !strings.Contains(got, "animate-pulse") {
		t.Errorf("Skeleton: expected animate-pulse class, got:\n%s", got)
	}
	if !strings.Contains(got, "bg-muted") {
		t.Errorf("Skeleton: expected bg-muted class, got:\n%s", got)
	}
}

func TestSkeleton_extraAttrs(t *testing.T) {
	got := testutil.RenderNode(t, core.Skeleton(g.Attr("data-test", "loading")))
	if !strings.Contains(got, `data-test="loading"`) {
		t.Errorf("Skeleton extra: expected data-test attribute, got:\n%s", got)
	}
}

func TestSkeleton_isDiv(t *testing.T) {
	got := testutil.RenderNode(t, core.Skeleton())
	if !strings.HasPrefix(got, "<div") {
		t.Errorf("Skeleton: expected <div> element, got:\n%s", got)
	}
}
