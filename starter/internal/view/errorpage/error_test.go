package errorpage_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/foundry/internal/view/errorpage"
	"github.com/go-sum/web"
	"github.com/go-sum/web/render"
)

const (
	_badgeClass = "inline-flex items-center justify-center rounded-md border px-2 py-0.5 text-xs font-medium w-fit whitespace-nowrap shrink-0 transition-colors border-transparent bg-secondary text-secondary-foreground"
	_btnClass   = "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-all disabled:pointer-events-none disabled:opacity-50 outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] cursor-pointer text-foreground underline-offset-4 hover:underline h-9 px-4 py-2"

	_cardOpen    = `<div class="max-w-lg mx-auto py-24 px-4"><div class="w-full rounded-lg border bg-card text-card-foreground shadow-xs"><div class="p-6">`
	_cardFooter  = `</div><div class="flex items-center p-6 pt-0"><a class="` + _btnClass + `" href="/">Back to home</a></div></div></div>`
)

func errorCard(status, title, body string) string {
	return _cardOpen +
		`<span class="` + _badgeClass + `">` + status + `</span>` +
		`<h1 class="text-2xl font-bold text-card-foreground mt-4 mb-3">` + title + `</h1>` +
		body +
		_cardFooter
}

func TestErrorContent_NotFound(t *testing.T) {
	t.Parallel()

	e := web.ErrNotFound("page not found")
	got := render.RenderNode(t, errorpage.ErrorContent(e))
	want := errorCard("404", "Not Found", `<p class="text-sm text-muted-foreground">page not found</p>`)

	if got != want {
		t.Fatalf("ErrorContent(ErrNotFound) mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestErrorContent_Forbidden(t *testing.T) {
	t.Parallel()

	e := web.ErrForbidden("access denied")
	got := render.RenderNode(t, errorpage.ErrorContent(e))
	want := errorCard("403", "Forbidden", `<p class="text-sm text-muted-foreground">access denied</p>`)

	if got != want {
		t.Fatalf("ErrorContent(ErrForbidden) mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestErrorContent_Internal_NoCauseLeak(t *testing.T) {
	t.Parallel()

	e := web.ErrInternal(errors.New("secret internal detail"))
	got := render.RenderNode(t, errorpage.ErrorContent(e))

	// Must not contain the internal cause string.
	if strings.Contains(got, "secret internal detail") {
		t.Fatalf("ErrorContent(ErrInternal) leaks cause text: %q", got)
	}

	// Must contain the generic retry message.
	if !strings.Contains(got, "Something went wrong. Please try again or contact support.") {
		t.Fatalf("ErrorContent(ErrInternal) missing generic retry message; got: %q", got)
	}

	want := errorCard("500", "Internal Server Error",
		`<p class="text-sm text-muted-foreground mb-2">Something went wrong. Please try again or contact support.</p>`)
	if got != want {
		t.Fatalf("ErrorContent(ErrInternal) mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestErrorContent_HTMLEntityEncoding(t *testing.T) {
	t.Parallel()

	e := web.ErrBadRequest("input has <special> & chars")
	got := render.RenderNode(t, errorpage.ErrorContent(e))

	// Entities must be encoded in the rendered output.
	if !strings.Contains(got, "input has &lt;special&gt; &amp; chars") {
		t.Fatalf("expected HTML entity encoding in output; got: %q", got)
	}

	// Raw unsafe characters must not appear.
	if strings.Contains(got, "<special>") {
		t.Fatalf("raw '<special>' must not appear in rendered HTML; got: %q", got)
	}

	want := errorCard("400", "Bad Request",
		`<p class="text-sm text-muted-foreground">input has &lt;special&gt; &amp; chars</p>`)
	if got != want {
		t.Fatalf("ErrorContent(ErrBadRequest special chars) mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestErrorContent_Internal_WithInstance(t *testing.T) {
	t.Parallel()

	e := web.ErrInternal(errors.New("db crash"))
	e.Instance = "/api/users/42"
	got := render.RenderNode(t, errorpage.ErrorContent(e))

	want := errorCard("500", "Internal Server Error",
		`<p class="text-sm text-muted-foreground mb-2">Something went wrong. Please try again or contact support.</p>`+
			`<p class="text-xs text-muted-foreground">Reference: /api/users/42</p>`)
	if got != want {
		t.Fatalf("ErrorContent(ErrInternal with instance) mismatch\ngot:  %s\nwant: %s", got, want)
	}

	// Instance must be present; cause must not be present.
	if strings.Contains(got, "db crash") {
		t.Fatalf("rendered HTML must not contain cause text 'db crash'; got: %q", got)
	}
	if !strings.Contains(got, "/api/users/42") {
		t.Fatalf("rendered HTML must contain instance '/api/users/42'; got: %q", got)
	}
}

func TestErrorContent_IsStable(t *testing.T) {
	t.Parallel()

	e := web.ErrNotFound("page not found")
	got1 := render.RenderNode(t, errorpage.ErrorContent(e))
	got2 := render.RenderNode(t, errorpage.ErrorContent(e))
	if got1 != got2 {
		t.Fatalf("ErrorContent is not stable across calls\nfirst:  %s\nsecond: %s", got1, got2)
	}
}

func TestErrorPage_WrapsContentInLayout(t *testing.T) {
	t.Parallel()

	req := view.Request{
		CSRFToken: "csrf-123",
		Nonce:     "nonce-123",
		Flash:     []string{"Saved"},
	}
	e := web.ErrNotFound("page not found")

	got := render.RenderNode(t, errorpage.ErrorPage(req, e))
	want := render.RenderNode(t, req.Page(e.Title, errorpage.ErrorContent(e)))
	if got != want {
		t.Fatalf("ErrorPage() = %q, want %q", got, want)
	}
}
