// Package render bridges gomponents to web.Response, providing helpers that
// render g.Node trees into HTML responses.
package render

import (
	"bytes"
	"net/http"

	"github.com/go-sum/web"
	g "maragu.dev/gomponents"
)

// Component renders a gomponents node as a 200 HTML response.
func Component(node g.Node) (web.Response, error) {
	return ComponentWithStatus(http.StatusOK, node)
}

// ComponentWithStatus renders a gomponents node with a custom status code.
// If rendering fails, logs the cause and returns a 500 Internal Server Error.
func ComponentWithStatus(status int, node g.Node) (web.Response, error) {
	var buf bytes.Buffer
	if err := node.Render(&buf); err != nil {
		return web.Response{}, web.ErrInternal(err)
	}
	return web.HTMLBytes(status, buf.Bytes()), nil
}

// Fragment renders a gomponents node as a 200 HTML fragment response.
// Semantically identical to Component — the distinction signals caller intent
// (fragment vs full page).
func Fragment(node g.Node) (web.Response, error) {
	return Component(node)
}

// FragmentWithStatus renders a fragment with a custom status code.
func FragmentWithStatus(status int, node g.Node) (web.Response, error) {
	return ComponentWithStatus(status, node)
}
