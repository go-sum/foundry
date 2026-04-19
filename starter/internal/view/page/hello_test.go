package page

import (
	"testing"

	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/web/render"
)

func TestHelloPage(t *testing.T) {
	req := view.Request{}
	got := render.RenderNode(t, HelloPage(req, "World"))

	want := `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Hello World</title>` +
		`<meta name="viewport" content="width=device-width, initial-scale=1">` +
		`<meta name="csrf-token" content="">` +
		`<meta name="htmx-config" content="{&#34;antiForgery&#34;:{&#34;headerName&#34;:&#34;X-CSRF-Token&#34;,&#34;parameterName&#34;:&#34;_csrf&#34;,&#34;token&#34;:&#34;&#34;}}">` +
		`<link rel="stylesheet" href="/css/app.css"></head>` +
		`<body><script src="/js/htmx.min.js" defer nonce=""></script>` +
		`<div id="flash" hx-swap-oob="true"></div>` +
		`<div class="min-h-screen bg-gray-50">` +
		`<div class="max-w-2xl mx-auto py-16 px-4">` +
		`<div id="greeting">` +
		`<div><h1 class="text-3xl font-bold text-gray-900 mb-4">Hello, World!</h1>` +
		`<p class="text-gray-600">This greeting was rendered server-side.</p></div>` +
		`</div>` +
		`<div class="mt-8"><label class="block text-sm font-medium text-gray-700 mb-2">Change name:</label>` +
		`<input type="text" name="name" value="World" class="border border-gray-300 rounded px-3 py-2" hx-get="/hello/greeting" hx-trigger="keyup changed delay:300ms" hx-target="#greeting" hx-swap="innerHTML" hx-include="this"></div>` +
		`<div class="mt-4"><a href="/" class="text-blue-600 hover:underline">Back to home</a></div>` +
		`</div></div></body></html>`

	if got != want {
		t.Errorf("HelloPage output mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestHelloPartial(t *testing.T) {
	got := render.RenderNode(t, HelloPartial("World"))

	want := `<div><h1 class="text-3xl font-bold text-gray-900 mb-4">Hello, World!</h1>` +
		`<p class="text-gray-600">This greeting was rendered server-side.</p></div>`

	if got != want {
		t.Errorf("HelloPartial output mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestHelloPartial_HTMLEntities(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "apostrophe",
			input: "O'Brien",
			want: `<div><h1 class="text-3xl font-bold text-gray-900 mb-4">Hello, O&#39;Brien!</h1>` +
				`<p class="text-gray-600">This greeting was rendered server-side.</p></div>`,
		},
		{
			name:  "ampersand",
			input: "AT&T",
			want: `<div><h1 class="text-3xl font-bold text-gray-900 mb-4">Hello, AT&amp;T!</h1>` +
				`<p class="text-gray-600">This greeting was rendered server-side.</p></div>`,
		},
		{
			name:  "angle brackets",
			input: "<script>",
			want: `<div><h1 class="text-3xl font-bold text-gray-900 mb-4">Hello, &lt;script&gt;!</h1>` +
				`<p class="text-gray-600">This greeting was rendered server-side.</p></div>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := render.RenderNode(t, HelloPartial(tc.input))
			if got != tc.want {
				t.Errorf("HelloPartial(%q) output mismatch\ngot:  %s\nwant: %s", tc.input, got, tc.want)
			}
		})
	}
}
