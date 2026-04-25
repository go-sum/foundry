package componentry_test

import (
	"testing"

	"github.com/go-sum/showcase/componentry"
	"github.com/go-sum/componentry/testutil"
)

func TestShowcase_renders(t *testing.T) {
	got := testutil.RenderNode(t, componentry.Showcase())
	if got == "" {
		t.Fatal("Showcase() rendered empty output")
	}
}
