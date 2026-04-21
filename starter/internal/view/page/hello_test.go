package page

import (
	"testing"

	"github.com/go-sum/componentry/interactive/theme"
	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/web/render"
)

const (
	_btnClass   = "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-all disabled:pointer-events-none disabled:opacity-50 outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] cursor-pointer text-foreground underline-offset-4 hover:underline h-9 px-4 py-2"
	_inputClass = "flex w-full rounded-md border border-input bg-transparent text-base shadow-xs transition-colors outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50 md:text-sm h-9 min-w-0 px-3 py-1"
)

func TestHelloPage(t *testing.T) {
	req := view.Request{}
	got := render.RenderNode(t, HelloPage(req, "World", "/hello/greeting", "/"))

	themeScript := render.RenderNode(t, theme.InitScript())
	const selectorScript = `<script src="/js/componentry.min.js" defer></script>`

	want := `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Hello World</title>` +
		`<meta name="viewport" content="width=device-width, initial-scale=1">` +
		`<meta name="csrf-token" content="">` +
		`<meta name="htmx-config" content="{&#34;includeIndicatorStyles&#34;:false,&#34;antiForgery&#34;:{&#34;headerName&#34;:&#34;X-CSRF-Token&#34;,&#34;parameterName&#34;:&#34;_csrf&#34;,&#34;token&#34;:&#34;&#34;}}">` +
		`<link rel="stylesheet" href="/css/app.css">` +
		themeScript + `</head>` +
		`<body class="bg-background text-foreground min-h-screen flex flex-col"><script src="/js/htmx.min.js" defer nonce=""></script>` +
		`<div id="flash" class="container mx-auto px-4 pt-4 grid gap-2" hx-swap-oob="true" aria-live="polite"></div>` +
		`<main class="container mx-auto px-4 py-6 flex-1">` +
		`<div class="max-w-2xl mx-auto py-16 px-4">` +
		`<div id="greeting">` +
		`<div><h1 class="text-2xl font-bold text-foreground mb-4">Hello, World!</h1>` +
		`<p class="text-muted-foreground">This greeting was rendered server-side.</p></div>` +
		`</div>` +
		`<div class="mt-8"><div class="grid gap-2">` +
		`<label class="text-sm font-medium leading-none inline-block" for="name">Change name:</label>` +
		`<input class="` + _inputClass + `" type="text" id="name" name="name" value="World" hx-get="/hello/greeting" hx-trigger="keyup changed delay:300ms" hx-target="#greeting" hx-swap="innerHTML" hx-include="this">` +
		`</div></div>` +
		`<div class="mt-4"><a class="` + _btnClass + `" href="/">Back to home</a></div>` +
		`</div>` +
		`</main>` +
		selectorScript + `</body></html>`

	if got != want {
		t.Errorf("HelloPage output mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestHelloPartial(t *testing.T) {
	got := render.RenderNode(t, HelloPartial("World"))

	want := `<div><h1 class="text-2xl font-bold text-foreground mb-4">Hello, World!</h1>` +
		`<p class="text-muted-foreground">This greeting was rendered server-side.</p></div>`

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
			want: `<div><h1 class="text-2xl font-bold text-foreground mb-4">Hello, O&#39;Brien!</h1>` +
				`<p class="text-muted-foreground">This greeting was rendered server-side.</p></div>`,
		},
		{
			name:  "ampersand",
			input: "AT&T",
			want: `<div><h1 class="text-2xl font-bold text-foreground mb-4">Hello, AT&amp;T!</h1>` +
				`<p class="text-muted-foreground">This greeting was rendered server-side.</p></div>`,
		},
		{
			name:  "angle brackets",
			input: "<script>",
			want: `<div><h1 class="text-2xl font-bold text-foreground mb-4">Hello, &lt;script&gt;!</h1>` +
				`<p class="text-muted-foreground">This greeting was rendered server-side.</p></div>`,
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
