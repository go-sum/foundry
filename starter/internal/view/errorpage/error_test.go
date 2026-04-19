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

func TestErrorContent_NotFound(t *testing.T) {
	t.Parallel()

	e := web.ErrNotFound("page not found")
	got := render.RenderNode(t, errorpage.ErrorContent(e))
	want := `<div class="max-w-lg mx-auto py-24 px-4"><div class="bg-white rounded-lg border border-gray-200 shadow-sm p-8"><div class="inline-flex items-center rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600 mb-4">404</div><h1 class="text-2xl font-bold text-gray-900 mb-3">Not Found</h1><p class="text-sm text-gray-600">page not found</p><div class="mt-8"><a href="/" class="text-sm text-blue-600 hover:underline">Back to home</a></div></div></div>`

	if got != want {
		t.Fatalf("ErrorContent(ErrNotFound) mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestErrorContent_Forbidden(t *testing.T) {
	t.Parallel()

	e := web.ErrForbidden("access denied")
	got := render.RenderNode(t, errorpage.ErrorContent(e))
	want := `<div class="max-w-lg mx-auto py-24 px-4"><div class="bg-white rounded-lg border border-gray-200 shadow-sm p-8"><div class="inline-flex items-center rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600 mb-4">403</div><h1 class="text-2xl font-bold text-gray-900 mb-3">Forbidden</h1><p class="text-sm text-gray-600">access denied</p><div class="mt-8"><a href="/" class="text-sm text-blue-600 hover:underline">Back to home</a></div></div></div>`

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

	// Must contain the 500 badge number in the expected rendering context.
	want := `<div class="max-w-lg mx-auto py-24 px-4"><div class="bg-white rounded-lg border border-gray-200 shadow-sm p-8"><div class="inline-flex items-center rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600 mb-4">500</div><h1 class="text-2xl font-bold text-gray-900 mb-3">Internal Server Error</h1><p class="text-sm text-gray-600 mb-2">Something went wrong. Please try again or contact support.</p><div class="mt-8"><a href="/" class="text-sm text-blue-600 hover:underline">Back to home</a></div></div></div>`
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

	want := `<div class="max-w-lg mx-auto py-24 px-4"><div class="bg-white rounded-lg border border-gray-200 shadow-sm p-8"><div class="inline-flex items-center rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600 mb-4">400</div><h1 class="text-2xl font-bold text-gray-900 mb-3">Bad Request</h1><p class="text-sm text-gray-600">input has &lt;special&gt; &amp; chars</p><div class="mt-8"><a href="/" class="text-sm text-blue-600 hover:underline">Back to home</a></div></div></div>`
	if got != want {
		t.Fatalf("ErrorContent(ErrBadRequest special chars) mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestErrorContent_Internal_WithInstance(t *testing.T) {
	t.Parallel()

	e := web.ErrInternal(errors.New("db crash"))
	e.Instance = "/api/users/42"
	got := render.RenderNode(t, errorpage.ErrorContent(e))

	want := `<div class="max-w-lg mx-auto py-24 px-4"><div class="bg-white rounded-lg border border-gray-200 shadow-sm p-8"><div class="inline-flex items-center rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600 mb-4">500</div><h1 class="text-2xl font-bold text-gray-900 mb-3">Internal Server Error</h1><p class="text-sm text-gray-600 mb-2">Something went wrong. Please try again or contact support.</p><p class="text-xs text-gray-400">Reference: /api/users/42</p><div class="mt-8"><a href="/" class="text-sm text-blue-600 hover:underline">Back to home</a></div></div></div>`
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
