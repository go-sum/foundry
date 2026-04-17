package htmx

import (
	"encoding/json"

	"github.com/go-sum/web"
)

// SetLocation sets HX-Location. Accepts a URL string or JSON-encoded object.
func SetLocation(resp *web.Response, value string) {
	resp.Headers.Set("HX-Location", value)
}

// SetPushURL sets HX-Push-Url to the given URL (or "false" to prevent push).
func SetPushURL(resp *web.Response, url string) {
	resp.Headers.Set("HX-Push-Url", url)
}

// SetReplaceURL sets HX-Replace-Url to the given URL (or "false" to prevent replacement).
func SetReplaceURL(resp *web.Response, url string) {
	resp.Headers.Set("HX-Replace-Url", url)
}

// SetRedirect sets HX-Redirect to a client-side redirect URL.
func SetRedirect(resp *web.Response, url string) {
	resp.Headers.Set("HX-Redirect", url)
}

// SetRefresh sets HX-Refresh: true to force a full page reload.
func SetRefresh(resp *web.Response) {
	resp.Headers.Set("HX-Refresh", "true")
}

// SetReswap sets HX-Reswap to control how the response is swapped.
// Valid values: innerHTML, outerHTML, beforebegin, afterbegin, beforeend, afterend, delete, none.
func SetReswap(resp *web.Response, strategy string) {
	resp.Headers.Set("HX-Reswap", strategy)
}

// SetRetarget sets HX-Retarget to a CSS selector overriding the target element.
func SetRetarget(resp *web.Response, selector string) {
	resp.Headers.Set("HX-Retarget", selector)
}

// SetReselect sets HX-Reselect to a CSS selector for choosing part of the response.
func SetReselect(resp *web.Response, selector string) {
	resp.Headers.Set("HX-Reselect", selector)
}

// SetTrigger sets HX-Trigger. name is the event name; payload is optional JSON data.
// If payload is nil, the header value is just the event name.
// If payload is non-nil, the header value is JSON: {"eventName": payload}.
func SetTrigger(resp *web.Response, name string, payload any) {
	resp.Headers.Set("HX-Trigger", triggerValue(name, payload))
}

// SetTriggerAfterSettle sets HX-Trigger-After-Settle (same semantics as SetTrigger).
func SetTriggerAfterSettle(resp *web.Response, name string, payload any) {
	resp.Headers.Set("HX-Trigger-After-Settle", triggerValue(name, payload))
}

// SetTriggerAfterSwap sets HX-Trigger-After-Swap (same semantics as SetTrigger).
func SetTriggerAfterSwap(resp *web.Response, name string, payload any) {
	resp.Headers.Set("HX-Trigger-After-Swap", triggerValue(name, payload))
}

// triggerValue returns the header string for a trigger event.
// If payload is nil, returns the event name. If payload is non-nil, returns
// JSON of the form {"name": payload}.
func triggerValue(name string, payload any) string {
	if payload == nil {
		return name
	}
	data, err := json.Marshal(map[string]any{name: payload})
	if err != nil {
		return name
	}
	return string(data)
}
