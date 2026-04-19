package htmx

import "net/http"

// Request contains HTMX request metadata extracted from HTTP headers.
type Request struct {
	Enabled     bool
	Boosted     bool
	Trigger     string
	Target      string
	TriggerName string
	CurrentURL  string
}

// Response contains HTMX response metadata written back as headers.
type Response struct {
	Redirect           string
	Refresh            bool
	PushURL            string
	ReplaceURL         string
	Trigger            string
	TriggerAfterSettle string
	TriggerAfterSwap   string
	Retarget           string
	Reswap             string
}

// NewRequest extracts HTMX request metadata from r.
func NewRequest(r *http.Request) Request {
	return Request{
		Enabled:     r.Header.Get("HX-Request") == "true",
		Boosted:     r.Header.Get("HX-Boosted") == "true",
		Trigger:     r.Header.Get("HX-Trigger"),
		Target:      r.Header.Get("HX-Target"),
		TriggerName: r.Header.Get("HX-Trigger-Name"),
		CurrentURL:  r.Header.Get("HX-Current-URL"),
	}
}

// IsPartial reports whether the request should receive a fragment instead of
// a full-page response. Boosted requests still expect a full page.
func (r Request) IsPartial() bool {
	return r.Enabled && !r.Boosted
}

// Request inspection helpers.

func IsRequest(r *http.Request) bool {
	return NewRequest(r).Enabled
}

func IsBoosted(r *http.Request) bool {
	return NewRequest(r).Boosted
}

func GetTrigger(r *http.Request) string {
	return NewRequest(r).Trigger
}

func GetTarget(r *http.Request) string {
	return NewRequest(r).Target
}

func GetTriggerName(r *http.Request) string {
	return NewRequest(r).TriggerName
}

func GetCurrentURL(r *http.Request) string {
	return NewRequest(r).CurrentURL
}

// Apply writes the response metadata to w as HTMX response headers.
func (resp Response) Apply(w http.ResponseWriter) {
	if resp.Redirect != "" {
		w.Header().Set("HX-Redirect", resp.Redirect)
	}
	if resp.Refresh {
		w.Header().Set("HX-Refresh", "true")
	}
	if resp.PushURL != "" {
		w.Header().Set("HX-Push-Url", resp.PushURL)
	}
	if resp.ReplaceURL != "" {
		w.Header().Set("HX-Replace-Url", resp.ReplaceURL)
	}
	if resp.Trigger != "" {
		w.Header().Set("HX-Trigger", resp.Trigger)
	}
	if resp.TriggerAfterSettle != "" {
		w.Header().Set("HX-Trigger-After-Settle", resp.TriggerAfterSettle)
	}
	if resp.TriggerAfterSwap != "" {
		w.Header().Set("HX-Trigger-After-Swap", resp.TriggerAfterSwap)
	}
	if resp.Retarget != "" {
		w.Header().Set("HX-Retarget", resp.Retarget)
	}
	if resp.Reswap != "" {
		w.Header().Set("HX-Reswap", resp.Reswap)
	}
}

func SetRedirect(w http.ResponseWriter, url string) {
	Response{Redirect: url}.Apply(w)
}

func SetRefresh(w http.ResponseWriter) {
	Response{Refresh: true}.Apply(w)
}

func SetPushURL(w http.ResponseWriter, url string) {
	Response{PushURL: url}.Apply(w)
}

func SetReplaceURL(w http.ResponseWriter, url string) {
	Response{ReplaceURL: url}.Apply(w)
}

func SetTrigger(w http.ResponseWriter, event string) {
	Response{Trigger: event}.Apply(w)
}

func SetTriggerAfterSettle(w http.ResponseWriter, event string) {
	Response{TriggerAfterSettle: event}.Apply(w)
}

func SetTriggerAfterSwap(w http.ResponseWriter, event string) {
	Response{TriggerAfterSwap: event}.Apply(w)
}

func SetRetarget(w http.ResponseWriter, selector string) {
	Response{Retarget: selector}.Apply(w)
}

func SetReswap(w http.ResponseWriter, strategy string) {
	Response{Reswap: strategy}.Apply(w)
}
