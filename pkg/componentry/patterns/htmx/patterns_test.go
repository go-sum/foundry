package htmx_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/go-sum/foundry/pkg/componentry/patterns/htmx"
	testutil "github.com/go-sum/foundry/pkg/componentry/testutil"
	"github.com/go-sum/foundry/pkg/componentry/ui/feedback"
)

// renderNodes wraps a slice of nodes in a div for rendering.
func renderNodes(t *testing.T, nodes []g.Node) string {
	t.Helper()
	return renderAttrs(t, nodes)
}

// ---- LiveSearch ----

func TestLiveSearch_defaults(t *testing.T) {
	got := renderNodes(t, htmx.LiveSearch(htmx.LiveSearchProps{
		Path:   "/search",
		Target: "#results",
	}))
	if !containsStr(got, `hx-get="/search"`) {
		t.Errorf("expected hx-get, got: %s", got)
	}
	if !containsStr(got, `hx-target="#results"`) {
		t.Errorf("expected hx-target, got: %s", got)
	}
	if !containsStr(got, `hx-swap="innerHTML"`) {
		t.Errorf("expected default hx-swap=innerHTML, got: %s", got)
	}
	if !containsStr(got, "300ms") {
		t.Errorf("expected default delay 300ms in trigger, got: %s", got)
	}
	if !containsStr(got, "input changed") {
		t.Errorf("expected 'input changed' trigger, got: %s", got)
	}
	if !containsStr(got, ", search") {
		t.Errorf("expected ', search' in trigger, got: %s", got)
	}
	if containsStr(got, "hx-push-url") {
		t.Errorf("expected no hx-push-url by default, got: %s", got)
	}
}

func TestLiveSearch_customDelay(t *testing.T) {
	got := renderNodes(t, htmx.LiveSearch(htmx.LiveSearchProps{
		Path:  "/search",
		Delay: "500ms",
	}))
	if !containsStr(got, "500ms") {
		t.Errorf("expected custom delay 500ms, got: %s", got)
	}
}

func TestLiveSearch_customTrigger(t *testing.T) {
	got := renderNodes(t, htmx.LiveSearch(htmx.LiveSearchProps{
		Path:    "/search",
		Trigger: "keyup changed delay:400ms",
	}))
	if !containsStr(got, `hx-trigger="keyup changed delay:400ms"`) {
		t.Errorf("expected custom trigger, got: %s", got)
	}
}

func TestLiveSearch_pushURL(t *testing.T) {
	got := renderNodes(t, htmx.LiveSearch(htmx.LiveSearchProps{
		Path:    "/search",
		PushURL: true,
	}))
	if !containsStr(got, `hx-push-url="true"`) {
		t.Errorf("expected hx-push-url=true, got: %s", got)
	}
}

func TestLiveSearch_customSwap(t *testing.T) {
	got := renderNodes(t, htmx.LiveSearch(htmx.LiveSearchProps{
		Path: "/search",
		Swap: htmx.SwapOuterHTML,
	}))
	if !containsStr(got, `hx-swap="outerHTML"`) {
		t.Errorf("expected custom hx-swap=outerHTML, got: %s", got)
	}
}

// ---- InlineValidation ----

func TestInlineValidation_defaults(t *testing.T) {
	got := renderNodes(t, htmx.InlineValidation(htmx.InlineValidationProps{
		Path:   "/validate/email",
		Target: "#email-error",
	}))
	if !containsStr(got, `hx-get="/validate/email"`) {
		t.Errorf("expected hx-get, got: %s", got)
	}
	if !containsStr(got, `hx-swap="outerHTML"`) {
		t.Errorf("expected default hx-swap=outerHTML, got: %s", got)
	}
	if !containsStr(got, `hx-trigger="change delay:200ms, blur"`) {
		t.Errorf("expected default trigger, got: %s", got)
	}
	if !containsStr(got, `hx-sync="closest form:abort"`) {
		t.Errorf("expected default hx-sync, got: %s", got)
	}
}

func TestInlineValidation_customTrigger(t *testing.T) {
	got := renderNodes(t, htmx.InlineValidation(htmx.InlineValidationProps{
		Path:    "/validate/name",
		Trigger: "blur",
	}))
	if !containsStr(got, `hx-trigger="blur"`) {
		t.Errorf("expected custom trigger, got: %s", got)
	}
}

func TestInlineValidation_customSync(t *testing.T) {
	got := renderNodes(t, htmx.InlineValidation(htmx.InlineValidationProps{
		Path: "/validate/name",
		Sync: "this:abort",
	}))
	if !containsStr(got, `hx-sync="this:abort"`) {
		t.Errorf("expected custom hx-sync, got: %s", got)
	}
}

// ---- PaginatedTableLink ----

func TestPaginatedTableLink_defaults(t *testing.T) {
	got := renderNodes(t, htmx.PaginatedTableLink(htmx.PaginatedTableProps{
		Path:   "/items",
		Page:   2,
		Target: "#table-region",
	}))
	if !containsStr(got, `hx-get="/items?page=2"`) {
		t.Errorf("expected hx-get with page param, got: %s", got)
	}
	if !containsStr(got, `hx-swap="outerHTML"`) {
		t.Errorf("expected default hx-swap=outerHTML, got: %s", got)
	}
	if containsStr(got, "hx-push-url") {
		t.Errorf("expected no hx-push-url by default, got: %s", got)
	}
}

func TestPaginatedTableLink_customPageParam(t *testing.T) {
	got := renderNodes(t, htmx.PaginatedTableLink(htmx.PaginatedTableProps{
		Path:      "/items",
		Page:      3,
		PageParam: "p",
	}))
	if !containsStr(got, "p=3") {
		t.Errorf("expected custom page param p=3, got: %s", got)
	}
	if containsStr(got, "page=") {
		t.Errorf("expected no 'page=' param, got: %s", got)
	}
}

func TestPaginatedTableLink_extraQuery(t *testing.T) {
	got := renderNodes(t, htmx.PaginatedTableLink(htmx.PaginatedTableProps{
		Path:  "/items",
		Page:  1,
		Query: map[string]string{"sort": "name"},
	}))
	if !containsStr(got, "sort=name") {
		t.Errorf("expected sort=name in URL, got: %s", got)
	}
	if !containsStr(got, "page=1") {
		t.Errorf("expected page=1 in URL, got: %s", got)
	}
}

func TestPaginatedTableLink_pushURL(t *testing.T) {
	got := renderNodes(t, htmx.PaginatedTableLink(htmx.PaginatedTableProps{
		Path:    "/items",
		Page:    1,
		PushURL: true,
	}))
	if !containsStr(got, `hx-push-url="true"`) {
		t.Errorf("expected hx-push-url=true, got: %s", got)
	}
}

// ---- AsyncDialogTrigger ----

func TestAsyncDialogTrigger_defaults(t *testing.T) {
	got := renderNodes(t, htmx.AsyncDialogTrigger(htmx.AsyncDialogProps{
		Path:     "/dialogs/confirm",
		DialogID: "confirm-dialog",
		Target:   "#confirm-dialog",
	}))
	if !containsStr(got, `data-dialog-open="confirm-dialog"`) {
		t.Errorf("expected data-dialog-open, got: %s", got)
	}
	if !containsStr(got, `aria-haspopup="dialog"`) {
		t.Errorf("expected aria-haspopup=dialog, got: %s", got)
	}
	if !containsStr(got, `aria-controls="confirm-dialog"`) {
		t.Errorf("expected aria-controls, got: %s", got)
	}
	if !containsStr(got, `hx-get="/dialogs/confirm"`) {
		t.Errorf("expected hx-get, got: %s", got)
	}
	if !containsStr(got, `hx-swap="innerHTML"`) {
		t.Errorf("expected default hx-swap=innerHTML, got: %s", got)
	}
}

func TestAsyncDialogTrigger_customSwap(t *testing.T) {
	got := renderNodes(t, htmx.AsyncDialogTrigger(htmx.AsyncDialogProps{
		Path:     "/dialogs/edit",
		DialogID: "edit-dialog",
		Swap:     htmx.SwapOuterHTML,
	}))
	if !containsStr(got, `hx-swap="outerHTML"`) {
		t.Errorf("expected custom hx-swap=outerHTML, got: %s", got)
	}
}

func TestAsyncDialogTrigger_withSelect(t *testing.T) {
	got := renderNodes(t, htmx.AsyncDialogTrigger(htmx.AsyncDialogProps{
		Path:     "/dialogs/view",
		DialogID: "view-dialog",
		Select:   "#dialog-body",
	}))
	if !containsStr(got, `hx-select="#dialog-body"`) {
		t.Errorf("expected hx-select, got: %s", got)
	}
}

// ---- OOBSwap ----

func TestOOBSwap_defaultStrategy(t *testing.T) {
	got := renderNodes(t, htmx.OOBSwap(htmx.OOBSwapProps{}))
	if !containsStr(got, `hx-swap-oob="true"`) {
		t.Errorf("expected hx-swap-oob=true, got: %s", got)
	}
}

func TestOOBSwap_withSelector(t *testing.T) {
	got := renderNodes(t, htmx.OOBSwap(htmx.OOBSwapProps{
		Selector: "#notifications",
	}))
	// When only Selector is set (Strategy defaults to "true"), it should become outerHTML
	if !containsStr(got, `hx-swap-oob="outerHTML:#notifications"`) {
		t.Errorf("expected hx-swap-oob=outerHTML:#notifications, got: %s", got)
	}
}

func TestOOBSwap_explicitStrategy(t *testing.T) {
	got := renderNodes(t, htmx.OOBSwap(htmx.OOBSwapProps{
		Strategy: htmx.SwapBeforeEnd,
		Selector: "#list",
	}))
	if !containsStr(got, `hx-swap-oob="beforeend:#list"`) {
		t.Errorf("expected hx-swap-oob=beforeend:#list, got: %s", got)
	}
}

// ---- OOBAppend ----

func TestOOBAppend(t *testing.T) {
	got := renderNodes(t, htmx.OOBAppend("#items"))
	if !containsStr(got, `hx-swap-oob="beforeend:#items"`) {
		t.Errorf("expected hx-swap-oob=beforeend:#items, got: %s", got)
	}
}

// ---- ToastOOB ----

func TestToastOOB_defaults(t *testing.T) {
	got := testutil.RenderNode(t, htmx.ToastOOB(htmx.ToastOOBProps{
		Toast: feedback.ToastProps{
			Title:   "Saved",
			Variant: feedback.ToastSuccess,
		},
	}))
	// Toast markup
	if !containsStr(got, "Saved") {
		t.Errorf("expected toast title 'Saved', got: %s", got)
	}
	// OOB swap for default #toast-container
	if !containsStr(got, `hx-swap-oob="beforeend:#toast-container"`) {
		t.Errorf("expected hx-swap-oob=beforeend:#toast-container, got: %s", got)
	}
}

func TestToastOOB_customSelector(t *testing.T) {
	got := testutil.RenderNode(t, htmx.ToastOOB(htmx.ToastOOBProps{
		Toast:    feedback.ToastProps{Title: "Notice"},
		Selector: "#alerts",
	}))
	if !containsStr(got, `hx-swap-oob="beforeend:#alerts"`) {
		t.Errorf("expected hx-swap-oob with custom selector, got: %s", got)
	}
}

func TestToastOOB_customStrategy(t *testing.T) {
	got := testutil.RenderNode(t, htmx.ToastOOB(htmx.ToastOOBProps{
		Toast:    feedback.ToastProps{Title: "Notice"},
		Selector: "#alerts",
		Strategy: htmx.SwapAfterEnd,
	}))
	if !containsStr(got, `hx-swap-oob="afterend:#alerts"`) {
		t.Errorf("expected hx-swap-oob with afterend strategy, got: %s", got)
	}
}

func TestToastOOB_preservesExtraNodes(t *testing.T) {
	extraNode := g.Attr("data-custom", "value")
	got := testutil.RenderNode(t, htmx.ToastOOB(htmx.ToastOOBProps{
		Toast: feedback.ToastProps{
			Title: "Hello",
			Extra: []g.Node{extraNode},
		},
	}))
	if !containsStr(got, `data-custom="value"`) {
		t.Errorf("expected extra node preserved, got: %s", got)
	}
	if !containsStr(got, "hx-swap-oob") {
		t.Errorf("expected hx-swap-oob, got: %s", got)
	}
}

// ---- DependentSelect ----

func TestDependentSelect_defaults(t *testing.T) {
	got := renderNodes(t, htmx.DependentSelect(htmx.DependentSelectProps{
		Path:   "/options/sub",
		Target: "#sub-select",
	}))
	if !containsStr(got, `hx-get="/options/sub"`) {
		t.Errorf("expected hx-get, got: %s", got)
	}
	if !containsStr(got, `hx-swap="outerHTML"`) {
		t.Errorf("expected default hx-swap=outerHTML, got: %s", got)
	}
	if !containsStr(got, `hx-trigger="change"`) {
		t.Errorf("expected default hx-trigger=change, got: %s", got)
	}
}

func TestDependentSelect_customTrigger(t *testing.T) {
	got := renderNodes(t, htmx.DependentSelect(htmx.DependentSelectProps{
		Path:    "/options/sub",
		Trigger: "click",
	}))
	if !containsStr(got, `hx-trigger="click"`) {
		t.Errorf("expected custom trigger, got: %s", got)
	}
}

// ---- InfiniteScroll ----

func TestInfiniteScroll_defaults(t *testing.T) {
	got := renderNodes(t, htmx.InfiniteScroll(htmx.InfiniteScrollProps{
		Path:   "/items?page=2",
		Target: "#item-list",
	}))
	if !containsStr(got, `hx-get="/items?page=2"`) {
		t.Errorf("expected hx-get, got: %s", got)
	}
	if !containsStr(got, `hx-swap="beforeend"`) {
		t.Errorf("expected default hx-swap=beforeend, got: %s", got)
	}
	if !containsStr(got, `hx-trigger="revealed"`) {
		t.Errorf("expected hx-trigger=revealed, got: %s", got)
	}
}

func TestInfiniteScroll_customSwap(t *testing.T) {
	got := renderNodes(t, htmx.InfiniteScroll(htmx.InfiniteScrollProps{
		Path: "/items?page=3",
		Swap: htmx.SwapAfterEnd,
	}))
	if !containsStr(got, `hx-swap="afterend"`) {
		t.Errorf("expected custom hx-swap=afterend, got: %s", got)
	}
	// Trigger must still be revealed
	if !containsStr(got, `hx-trigger="revealed"`) {
		t.Errorf("expected hx-trigger=revealed, got: %s", got)
	}
}

func TestInfiniteScroll_withSelect(t *testing.T) {
	got := renderNodes(t, htmx.InfiniteScroll(htmx.InfiniteScrollProps{
		Path:   "/items?page=4",
		Select: ".item-row",
	}))
	if !containsStr(got, `hx-select=".item-row"`) {
		t.Errorf("expected hx-select, got: %s", got)
	}
}

// ---- FormSubmit ----

func TestFormSubmit_defaults(t *testing.T) {
	got := renderNodes(t, htmx.FormSubmit(htmx.FormSubmitProps{
		Path:   "/submit",
		Target: "#form-region",
	}))
	if !containsStr(got, `hx-post="/submit"`) {
		t.Errorf("expected hx-post, got: %s", got)
	}
	if !containsStr(got, `hx-swap="outerHTML"`) {
		t.Errorf("expected default hx-swap=outerHTML, got: %s", got)
	}
	if !containsStr(got, `hx-disabled-elt="this"`) {
		t.Errorf("expected default hx-disabled-elt=this, got: %s", got)
	}
	if containsStr(got, "hx-push-url") {
		t.Errorf("expected no hx-push-url by default, got: %s", got)
	}
}

func TestFormSubmit_customDisabledElt(t *testing.T) {
	got := renderNodes(t, htmx.FormSubmit(htmx.FormSubmitProps{
		Path:        "/submit",
		DisabledElt: "#submit-btn",
	}))
	if !containsStr(got, `hx-disabled-elt="#submit-btn"`) {
		t.Errorf("expected custom hx-disabled-elt, got: %s", got)
	}
}

func TestFormSubmit_pushURL(t *testing.T) {
	got := renderNodes(t, htmx.FormSubmit(htmx.FormSubmitProps{
		Path:    "/submit",
		PushURL: true,
	}))
	if !containsStr(got, `hx-push-url="true"`) {
		t.Errorf("expected hx-push-url=true, got: %s", got)
	}
}

func TestFormSubmit_encoding(t *testing.T) {
	got := renderNodes(t, htmx.FormSubmit(htmx.FormSubmitProps{
		Path:     "/upload",
		Encoding: "multipart/form-data",
	}))
	if !containsStr(got, `hx-encoding="multipart/form-data"`) {
		t.Errorf("expected hx-encoding, got: %s", got)
	}
}

// ---- withQueryParam (tested via PaginatedTableLink) ----

func TestWithQueryParam_appendsToExistingQuery(t *testing.T) {
	got := renderNodes(t, htmx.PaginatedTableLink(htmx.PaginatedTableProps{
		Path:  "/items?sort=asc",
		Page:  5,
		Query: map[string]string{"filter": "active"},
	}))
	if !containsStr(got, "sort=asc") {
		t.Errorf("expected existing query param preserved, got: %s", got)
	}
	if !containsStr(got, "filter=active") {
		t.Errorf("expected extra query param, got: %s", got)
	}
	if !containsStr(got, "page=5") {
		t.Errorf("expected page param, got: %s", got)
	}
}

func TestWithQueryParam_pageParamOverridesExisting(t *testing.T) {
	got := renderNodes(t, htmx.PaginatedTableLink(htmx.PaginatedTableProps{
		Path: "/items?page=1",
		Page: 7,
	}))
	if !containsStr(got, "page=7") {
		t.Errorf("expected page param to be overridden to 7, got: %s", got)
	}
	if strings.Count(got, "page=") > 1 {
		t.Errorf("expected page param to appear only once, got: %s", got)
	}
}
