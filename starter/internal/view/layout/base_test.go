package layout

import (
	"testing"

	"github.com/go-sum/web/render"

	g "maragu.dev/gomponents"
)

func TestPage(t *testing.T) {
	props := Props{
		Title:     "Test Page",
		Nonce:     "abc123",
		CSRFToken: "tok-csrf",
		Flash:     []string{"saved"},
		Children:  []g.Node{g.Text("content")},
	}
	got := render.RenderNode(t, Page(props))

	want := `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Test Page</title>` +
		`<meta name="viewport" content="width=device-width, initial-scale=1">` +
		`<meta name="csrf-token" content="tok-csrf">` +
		`<meta name="htmx-config" content="{&#34;antiForgery&#34;:{&#34;headerName&#34;:&#34;X-CSRF-Token&#34;,&#34;parameterName&#34;:&#34;_csrf&#34;,&#34;token&#34;:&#34;tok-csrf&#34;}}">` +
		`<link rel="stylesheet" href="/css/app.css"></head>` +
		`<body><script src="/js/htmx.min.js" defer nonce="abc123"></script>` +
		`<div id="flash" hx-swap-oob="true"><div>saved</div></div>` +
		`<div class="min-h-screen bg-gray-50">content</div></body></html>`

	if got != want {
		t.Errorf("Page output mismatch\ngot:  %s\nwant: %s", got, want)
	}
}
