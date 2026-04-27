package htmx

import (
	"strings"

	"github.com/go-sum/foundry/pkg/web"
)

// IsHTMX reports whether the request was made by HTMX (HX-Request: true).
func IsHTMX(c *web.Context) bool {
	return strings.EqualFold(c.Headers().Get("HX-Request"), "true")
}

// IsBoosted reports whether the request used hx-boost (HX-Boosted: true).
func IsBoosted(c *web.Context) bool {
	return strings.EqualFold(c.Headers().Get("HX-Boosted"), "true")
}

// HistoryRestore reports whether this is a history restore request (HX-History-Restore-Request: true).
func HistoryRestore(c *web.Context) bool {
	return strings.EqualFold(c.Headers().Get("HX-History-Restore-Request"), "true")
}

// CurrentURL returns the current URL of the browser (HX-Current-URL header value).
func CurrentURL(c *web.Context) string {
	return c.Headers().Get("HX-Current-URL")
}

// Prompt returns the user-provided value from an hx-prompt (HX-Prompt header value).
func Prompt(c *web.Context) string {
	return c.Headers().Get("HX-Prompt")
}

// Target returns the id of the element that triggered the swap (HX-Target header value).
func Target(c *web.Context) string {
	return c.Headers().Get("HX-Target")
}

// Trigger returns the id of the element that triggered the request (HX-Trigger header value).
func Trigger(c *web.Context) string {
	return c.Headers().Get("HX-Trigger")
}

// TriggerName returns the name of the element that triggered the request (HX-Trigger-Name header value).
func TriggerName(c *web.Context) string {
	return c.Headers().Get("HX-Trigger-Name")
}
