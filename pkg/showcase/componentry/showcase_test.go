package componentry_test

import (
	"testing"

	"github.com/go-sum/foundry/pkg/showcase/componentry"
	"github.com/go-sum/foundry/pkg/showcase/componentry/demo"
	"github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestShowcase_renders(t *testing.T) {
	paths := demo.NewPaths(componentry.DefaultConfig().BasePath)
	got := testutil.RenderNode(t, componentry.Showcase(nil, paths))
	if got == "" {
		t.Fatal("Showcase() rendered empty output")
	}
}
