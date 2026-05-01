package email_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/foundry/pkg/componentry/email"
	"github.com/go-sum/foundry/pkg/componentry/testutil"
)

func TestEmail(t *testing.T) {
	tests := []struct {
		name string
		node g.Node
	}{
		{
			name: "layout with title",
			node: email.Layout(email.LayoutProps{Title: "My Email"}, email.P("body")),
		},
		{
			name: "layout zero value",
			node: email.Layout(email.LayoutProps{}, email.P("content")),
		},
		{
			name: "layout custom bg color",
			node: email.Layout(email.LayoutProps{BgColor: "#123456"}, email.P("x")),
		},
		{
			name: "layout custom content width",
			node: email.Layout(email.LayoutProps{ContentWidth: 480}, email.P("x")),
		},
		{
			name: "layout with footer",
			node: email.Layout(email.LayoutProps{Footer: email.P("Footer line")}, email.P("body")),
		},
		{
			name: "h1",
			node: email.H1("Big Heading"),
		},
		{
			name: "h2",
			node: email.H2("Section"),
		},
		{
			name: "p",
			node: email.P("A paragraph."),
		},
		{
			name: "button",
			node: email.Button("Click me", "https://example.com"),
		},
		{
			name: "a",
			node: email.A("Click here", "https://example.com"),
		},
		{
			name: "hr",
			node: email.HR(),
		},
		{
			name: "preview text",
			node: email.PreviewText("Check this out"),
		},
		{
			name: "preview text xss safety",
			node: email.PreviewText("<script>alert('xss')</script>"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := testutil.RenderNode(t, tc.node)
			want := testutil.LoadGolden(t)
			testutil.AssertEqualHTML(t, want, got)
		})
	}
}

// TestPlainText verifies the string-returning PlainText helper.
// It does not produce HTML nodes so golden-file assertions do not apply.
func TestPlainText(t *testing.T) {
	t.Run("joins CRLF", func(t *testing.T) {
		result := email.PlainText("Line 1", "Line 2", "Line 3")
		if !strings.Contains(result, "\r\n") {
			t.Errorf("expected CRLF line endings, got: %q", result)
		}
	})

	t.Run("single line", func(t *testing.T) {
		result := email.PlainText("Hello")
		if result != "Hello" {
			t.Errorf("expected %q, got %q", "Hello", result)
		}
	})

	t.Run("empty lines", func(t *testing.T) {
		result := email.PlainText("A", "", "B")
		expected := "A\r\n\r\nB"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}
