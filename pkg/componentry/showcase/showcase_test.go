package showcase_test

import (
	"testing"

	"github.com/go-sum/componentry/showcase"
	"github.com/go-sum/componentry/testutil"
)

func TestShowcase_renders(t *testing.T) {
	got := testutil.RenderNode(t, showcase.Showcase())
	if got == "" {
		t.Fatal("Showcase() rendered empty output")
	}
}
