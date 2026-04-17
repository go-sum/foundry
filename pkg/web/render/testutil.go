package render

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"
)

// RenderNode renders a gomponents node to a string for test assertions.
func RenderNode(t *testing.T, node g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := node.Render(&b); err != nil {
		t.Fatalf("render node: %v", err)
	}
	return b.String()
}
