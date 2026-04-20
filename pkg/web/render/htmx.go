package render

import (
	"strings"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// HXForm returns an HTMX-enhanced form element.
// action is the hx-post target URL; children can include inputs, buttons, CSRFField, etc.
//
// The form uses hx-post and hx-swap="outerHTML" by default, matching the most common
// HTMX pattern (swap the form with the response fragment).
func HXForm(action string, children ...g.Node) g.Node {
	attrs := []g.Node{
		g.Attr("hx-post", action),
		g.Attr("hx-swap", "outerHTML"),
	}
	return h.FormEl(append(attrs, children...)...)
}

// HXGet returns an element with hx-get set to the given URL.
func HXGet(url string) g.Node {
	return g.Attr("hx-get", url)
}

// HXTarget returns an hx-target attribute pointing at a CSS selector.
func HXTarget(selector string) g.Node {
	return g.Attr("hx-target", selector)
}

// HXSwap returns an hx-swap attribute with the given swap strategy.
// Common values: innerHTML, outerHTML, beforebegin, afterbegin, beforeend, afterend, delete, none.
func HXSwap(strategy string) g.Node {
	return g.Attr("hx-swap", strategy)
}

// HXTrigger returns an hx-trigger attribute.
func HXTrigger(trigger string) g.Node {
	return g.Attr("hx-trigger", trigger)
}

// HXBoost returns an hx-boost="true" attribute to boost anchor/form navigation.
func HXBoost() g.Node {
	return g.Attr("hx-boost", "true")
}

// HXPushURL returns an hx-push-url attribute.
func HXPushURL(url string) g.Node {
	return g.Attr("hx-push-url", url)
}

// HXCSRFHeaders returns an hx-headers attribute that injects the CSRF token
// into all HTMX requests from the element and its children. Apply to <body>
// or the root HTMX container:
//
//	g.Body(render.HXCSRFHeaders(secure.CSRFToken(c)), ...)
//
// This produces: hx-headers="{\"X-CSRF-Token\":\"<token>\"}"
// which HTMX includes as a request header on every triggered request.
func HXCSRFHeaders(token string) g.Node {
	return g.Attr("hx-headers", jsonCSRFHeader(token))
}

// HXCSRFMeta returns a <meta name="htmx-config"> element that configures
// HTMX globally with the CSRF token via antiForgery settings. Place in <head>.
// This is an alternative to HXCSRFHeaders for apps that cannot easily add
// attributes to the HTMX root element.
//
// includeIndicatorStyles is disabled to prevent HTMX from injecting a runtime <style> element,
// which would violate Content-Security-Policy. Indicator styles are provided by the static CSS bundle.
//
// Produces:
//
//	<meta name="htmx-config" content='{"includeIndicatorStyles":false,"antiForgery":{"headerName":"X-CSRF-Token","parameterName":"_csrf","token":"<token>"}}'>
func HXCSRFMeta(token string) g.Node {
	content := `{"includeIndicatorStyles":false,"antiForgery":{"headerName":"X-CSRF-Token","parameterName":"_csrf","token":"` + jsonEscapeToken(token) + `"}}`
	return g.El("meta",
		g.Attr("name", "htmx-config"),
		g.Attr("content", content),
	)
}

func jsonCSRFHeader(token string) string {
	return `{"X-CSRF-Token":"` + jsonEscapeToken(token) + `"}`
}

// jsonEscapeToken escapes special characters in a CSRF token for safe
// embedding in a JSON string. CSRF tokens are base64url so this mainly
// guards against unexpected values.
func jsonEscapeToken(token string) string {
	// Replace the few characters that matter in JSON strings.
	// CSRF tokens from this library are base64url (A-Z a-z 0-9 - _ =),
	// so this is defensive-only.
	token = strings.ReplaceAll(token, `\`, `\\`)
	token = strings.ReplaceAll(token, `"`, `\"`)
	return token
}
