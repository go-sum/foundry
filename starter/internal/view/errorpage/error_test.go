package errorpage_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-sum/foundry/internal/view"
	"github.com/go-sum/foundry/internal/view/errorpage"
	"github.com/go-sum/web"
	"github.com/go-sum/web/render"
)

const (
	_badgeClass = "inline-flex items-center justify-center rounded-md border px-2 py-0.5 text-xs font-medium w-fit whitespace-nowrap shrink-0 transition-colors border-transparent bg-secondary text-secondary-foreground"
	_btnClass   = "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-all disabled:pointer-events-none disabled:opacity-50 outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] cursor-pointer bg-primary text-primary-foreground shadow-xs hover:bg-primary/90 h-9 px-4 py-2"

	_alertDefault     = `class="relative w-full rounded-lg border px-4 py-3 text-sm grid gap-1.5 items-start backdrop-blur-sm border-primary/30 bg-primary/20 text-primary [&amp;_[data-alert-description]]:text-muted-foreground" role="alert" aria-live="polite"`
	_alertDestructive = `class="relative w-full rounded-lg border px-4 py-3 text-sm grid gap-1.5 items-start backdrop-blur-sm border-destructive/30 bg-destructive/20 text-destructive [&amp;_[data-alert-description]]:text-destructive/80" role="alert" aria-live="assertive"`
	_alertDesc        = `class="grid justify-items-start gap-1 text-sm" data-alert-description=""`

	_cardOpen  = `<div class="mx-auto flex max-w-2xl flex-col gap-6 py-16"><div class="w-full rounded-lg border bg-card text-card-foreground shadow-xs">`
	_cardClose = `</div></div>`
)

func cardHeader(status int, title string) string {
	return `<div class="flex flex-col space-y-1.5 p-6 pb-0">` +
		`<div class="flex items-start justify-between gap-4">` +
		`<div class="space-y-1">` +
		`<h3 class="text-lg font-semibold leading-none tracking-tight">` + title + `</h3>` +
		`<p class="text-sm text-muted-foreground">` + fmt.Sprintf("%d %s", status, title) + `</p>` +
		`</div>` +
		`<span class="` + _badgeClass + `">HTTP ` + fmt.Sprintf("%d", status) + `</span>` +
		`</div></div>`
}

func cardContent(alertAttrs, message, extra string) string {
	return `<div class="p-6"><div class="space-y-4">` +
		`<div ` + alertAttrs + `><div ` + _alertDesc + `>` + message + `</div></div>` +
		`<div class="flex flex-wrap gap-3"><a class="` + _btnClass + `" href="/">Return Home</a></div>` +
		extra +
		`</div></div>`
}

func TestErrorContent_NotFound(t *testing.T) {
	t.Parallel()

	e := web.ErrNotFound("page not found")
	got := render.RenderNode(t, errorpage.ErrorContent(e))
	want := _cardOpen + cardHeader(404, "Not Found") + cardContent(_alertDefault, "page not found", "") + _cardClose

	if got != want {
		t.Fatalf("ErrorContent(ErrNotFound) mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestErrorContent_Forbidden(t *testing.T) {
	t.Parallel()

	e := web.ErrForbidden("access denied")
	got := render.RenderNode(t, errorpage.ErrorContent(e))
	want := _cardOpen + cardHeader(403, "Forbidden") + cardContent(_alertDefault, "access denied", "") + _cardClose

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

	want := _cardOpen + cardHeader(500, "Internal Server Error") + cardContent(_alertDestructive, "Something went wrong. Please try again or contact support.", "") + _cardClose
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

	want := _cardOpen + cardHeader(400, "Bad Request") + cardContent(_alertDefault, "input has &lt;special&gt; &amp; chars", "") + _cardClose
	if got != want {
		t.Fatalf("ErrorContent(ErrBadRequest special chars) mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestErrorContent_Internal_WithInstance(t *testing.T) {
	t.Parallel()

	e := web.ErrInternal(errors.New("db crash"))
	e.Instance = "/api/users/42"
	got := render.RenderNode(t, errorpage.ErrorContent(e))

	instance := `<p class="text-xs text-muted-foreground">Reference: /api/users/42</p>`
	want := _cardOpen + cardHeader(500, "Internal Server Error") + cardContent(_alertDestructive, "Something went wrong. Please try again or contact support.", instance) + _cardClose
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
