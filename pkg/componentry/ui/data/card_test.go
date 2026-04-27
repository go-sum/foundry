package data_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/foundry/pkg/componentry/ui/data"
	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestCard(t *testing.T) {
	t.Run("full composition", func(t *testing.T) {
		got := testutil.RenderNode(t, data.Card.Root(
			data.Card.Header(
				data.Card.Title(g.Text("Account Settings")),
				data.Card.Description(g.Text("Manage your account details.")),
			),
			data.Card.Content(g.Text("Content goes here.")),
			data.Card.Footer(g.Text("Footer content.")),
		))
		want := testutil.LoadGolden(t)
		testutil.AssertEqualHTML(t, want, got)
	})

	t.Run("empty body", func(t *testing.T) {
		got := testutil.RenderNode(t, data.Card.Root())
		want := testutil.LoadGolden(t)
		testutil.AssertEqualHTML(t, want, got)
	})

	t.Run("content only", func(t *testing.T) {
		got := testutil.RenderNode(t, data.Card.Root(
			data.Card.Content(g.Text("Just content.")),
		))
		want := testutil.LoadGolden(t)
		testutil.AssertEqualHTML(t, want, got)
	})

	t.Run("zero value no children", func(t *testing.T) {
		got := testutil.RenderNode(t, data.Card.Root())
		want := testutil.LoadGolden(t)
		testutil.AssertEqualHTML(t, want, got)
	})
}
