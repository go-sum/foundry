package htmx

import (
	"net/url"
	"strconv"

	"github.com/go-sum/foundry/pkg/componentry/ui/feedback"

	g "maragu.dev/gomponents"
)

// Swap strategy constants.
const (
	SwapInnerHTML   = "innerHTML"
	SwapOuterHTML   = "outerHTML"
	SwapBeforeEnd   = "beforeend"
	SwapAfterEnd    = "afterend"
	SwapBeforeBegin = "beforebegin"
	SwapDelete      = "delete"
	SwapNone        = "none"
)

// LiveSearchProps configures an input that fetches server-rendered results as the user types.
type LiveSearchProps struct {
	Path        string
	Target      string
	Swap        string
	Trigger     string
	Delay       string
	Include     string
	Indicator   string
	DisabledElt string
	PushURL     bool
}

// LiveSearch returns hx-* attributes for a debounced live-search input.
func LiveSearch(p LiveSearchProps) []g.Node {
	trigger := p.Trigger
	if trigger == "" {
		delay := orDefault(p.Delay, "300ms")
		trigger = "input changed delay:" + delay + ", search"
	}

	props := AttrsProps{
		Get:         p.Path,
		Target:      p.Target,
		Swap:        orDefault(p.Swap, SwapInnerHTML),
		Trigger:     trigger,
		Include:     p.Include,
		Indicator:   p.Indicator,
		DisabledElt: p.DisabledElt,
	}
	if p.PushURL {
		props.PushURL = "true"
	}
	return Attrs(props)
}

// InlineValidationProps configures a field that validates server-side on change/blur.
type InlineValidationProps struct {
	Path        string
	Target      string
	Swap        string
	Trigger     string
	Include     string
	Indicator   string
	DisabledElt string
	Sync        string
}

// InlineValidation returns hx-* attributes for a field that validates on blur.
func InlineValidation(p InlineValidationProps) []g.Node {
	trigger := orDefault(p.Trigger, "change delay:200ms, blur")
	sync := orDefault(p.Sync, "closest form:abort")

	return Attrs(AttrsProps{
		Get:         p.Path,
		Target:      p.Target,
		Swap:        orDefault(p.Swap, SwapOuterHTML),
		Trigger:     trigger,
		Include:     p.Include,
		Indicator:   p.Indicator,
		DisabledElt: p.DisabledElt,
		Sync:        sync,
	})
}

// PaginatedTableProps configures a link or button that swaps a server-rendered table region.
type PaginatedTableProps struct {
	Path        string
	Page        int
	PageParam   string
	Query       map[string]string
	Target      string
	Swap        string
	Include     string
	Indicator   string
	DisabledElt string
	PushURL     bool
}

// PaginatedTableLink returns hx-* attributes for a paginated navigation link.
func PaginatedTableLink(p PaginatedTableProps) []g.Node {
	path := withQueryParam(p.Path, orDefault(p.PageParam, "page"), strconv.Itoa(p.Page), p.Query)
	props := AttrsProps{
		Get:         path,
		Target:      p.Target,
		Swap:        orDefault(p.Swap, SwapOuterHTML),
		Include:     p.Include,
		Indicator:   p.Indicator,
		DisabledElt: p.DisabledElt,
	}
	if p.PushURL {
		props.PushURL = "true"
	}
	return Attrs(props)
}

// AsyncDialogProps configures a trigger that opens a native dialog and fetches its body asynchronously.
type AsyncDialogProps struct {
	Path        string
	DialogID    string
	Target      string
	Swap        string
	Select      string
	Indicator   string
	DisabledElt string
}

// AsyncDialogTrigger returns attributes for a trigger that loads a dialog via hx-get.
func AsyncDialogTrigger(p AsyncDialogProps) []g.Node {
	nodes := []g.Node{
		g.Attr("data-dialog-open", p.DialogID),
		g.Attr("aria-haspopup", "dialog"),
		g.Attr("aria-controls", p.DialogID),
	}
	nodes = append(nodes, Attrs(AttrsProps{
		Get:         p.Path,
		Target:      p.Target,
		Swap:        orDefault(p.Swap, SwapInnerHTML),
		Select:      p.Select,
		Indicator:   p.Indicator,
		DisabledElt: p.DisabledElt,
	})...)
	return nodes
}

// OOBSwapProps configures an out-of-band swap attribute.
type OOBSwapProps struct {
	Strategy string
	Selector string
}

// OOBSwap returns an hx-swap-oob attribute for out-of-band swapping.
func OOBSwap(p OOBSwapProps) []g.Node {
	value := orDefault(p.Strategy, "true")
	if p.Selector != "" {
		if value == "true" {
			value = SwapOuterHTML
		}
		value += ":" + p.Selector
	}
	return []g.Node{g.Attr("hx-swap-oob", value)}
}

// OOBAppend is a shortcut for OOBSwap with SwapBeforeEnd strategy.
func OOBAppend(selector string) []g.Node {
	return OOBSwap(OOBSwapProps{Strategy: SwapBeforeEnd, Selector: selector})
}

// ToastOOBProps wraps a feedback.Toast for out-of-band insertion into a toast container.
type ToastOOBProps struct {
	Toast    feedback.ToastProps
	Selector string
	Strategy string
}

// ToastOOB renders a toast node with OOB swap attributes for injection into a container.
func ToastOOB(p ToastOOBProps) g.Node {
	toast := p.Toast
	selector := orDefault(p.Selector, "#toast-container")
	extra := append([]g.Node{}, OOBSwap(OOBSwapProps{
		Strategy: orDefault(p.Strategy, SwapBeforeEnd),
		Selector: selector,
	})...)
	toast.Extra = append(extra, toast.Extra...)
	return feedback.Toast(toast)
}

// DependentSelectProps configures a select that swaps a downstream field when its value changes.
type DependentSelectProps struct {
	Path        string
	Target      string
	Swap        string
	Trigger     string
	Include     string
	Indicator   string
	DisabledElt string
}

// DependentSelect returns hx-* attributes for a cascading select element.
func DependentSelect(p DependentSelectProps) []g.Node {
	trigger := orDefault(p.Trigger, "change")
	return Attrs(AttrsProps{
		Get:         p.Path,
		Target:      p.Target,
		Swap:        orDefault(p.Swap, SwapOuterHTML),
		Trigger:     trigger,
		Include:     p.Include,
		Indicator:   p.Indicator,
		DisabledElt: p.DisabledElt,
	})
}

// InfiniteScrollProps configures an element that appends more content when revealed in the viewport.
type InfiniteScrollProps struct {
	Path      string
	Target    string
	Swap      string
	Select    string
	Indicator string
}

// InfiniteScroll returns hx-* attributes for append-on-reveal infinite scrolling.
func InfiniteScroll(p InfiniteScrollProps) []g.Node {
	return Attrs(AttrsProps{
		Get:       p.Path,
		Target:    p.Target,
		Swap:      orDefault(p.Swap, SwapBeforeEnd),
		Select:    p.Select,
		Indicator: p.Indicator,
		Trigger:   "revealed",
	})
}

// FormSubmitProps configures a form that disables itself on submit.
type FormSubmitProps struct {
	Path        string
	Target      string
	Swap        string
	Include     string
	Indicator   string
	DisabledElt string
	Encoding    string
	PushURL     bool
}

// FormSubmit returns hx-* attributes for a form with disable-on-submit behavior.
func FormSubmit(p FormSubmitProps) []g.Node {
	props := AttrsProps{
		Post:        p.Path,
		Target:      p.Target,
		Swap:        orDefault(p.Swap, SwapOuterHTML),
		Include:     p.Include,
		Indicator:   p.Indicator,
		DisabledElt: orDefault(p.DisabledElt, "this"),
		Encoding:    p.Encoding,
	}
	if p.PushURL {
		props.PushURL = "true"
	}
	return Attrs(props)
}

// withQueryParam builds a URL from path, setting key=value and merging extras into the query string.
func withQueryParam(path, key, value string, extras map[string]string) string {
	parsed, err := url.Parse(path)
	if err != nil {
		return path
	}
	query := parsed.Query()
	for name, extra := range extras {
		query.Set(name, extra)
	}
	query.Set(key, value)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

// orDefault returns value when non-empty, otherwise fallback.
func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
