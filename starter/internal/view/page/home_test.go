package page

import (
	"testing"

	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/web/render"
)

func TestHomePage(t *testing.T) {
	req := view.Request{}
	got := render.RenderNode(t, HomePage(req))

	want := `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Home</title>` +
		`<meta name="viewport" content="width=device-width, initial-scale=1">` +
		`<meta name="csrf-token" content="">` +
		`<meta name="htmx-config" content="{&#34;antiForgery&#34;:{&#34;headerName&#34;:&#34;X-CSRF-Token&#34;,&#34;parameterName&#34;:&#34;_csrf&#34;,&#34;token&#34;:&#34;&#34;}}">` +
		`<link rel="stylesheet" href="/css/app.css"></head>` +
		`<body><script src="/js/htmx.min.js" defer nonce=""></script>` +
		`<div id="flash" hx-swap-oob="true"></div>` +
		`<div class="min-h-screen bg-gray-50">` +
		`<div class="max-w-2xl mx-auto py-16 px-4">` +
		`<h1 class="text-3xl font-bold text-gray-900 mb-4">Welcome to Foundry</h1>` +
		`<p class="text-gray-600 mb-8">A Go web application built on W3C Web API primitives.</p>` +
		`<a href="/hello/World" class="text-blue-600 hover:underline">Say hello to World</a>` +
		`</div></div></body></html>`

	if got != want {
		t.Errorf("HomePage output mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestHomeContent(t *testing.T) {
	got := render.RenderNode(t, HomeContent())

	want := `<div class="max-w-2xl mx-auto py-16 px-4">` +
		`<h1 class="text-3xl font-bold text-gray-900 mb-4">Welcome to Foundry</h1>` +
		`<p class="text-gray-600 mb-8">A Go web application built on W3C Web API primitives.</p>` +
		`<a href="/hello/World" class="text-blue-600 hover:underline">Say hello to World</a>` +
		`</div>`

	if got != want {
		t.Errorf("HomeContent output mismatch\ngot:  %s\nwant: %s", got, want)
	}
}
