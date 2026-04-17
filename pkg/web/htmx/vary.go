package htmx

import "github.com/go-sum/web"

// VaryMiddleware returns middleware that adds "HX-Request" to the Vary response header,
// ensuring HTMX partial responses are not served from cache as full-page responses.
func VaryMiddleware() web.Middleware {
	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			resp, err := next(c)
			resp.Headers.Append("Vary", "HX-Request")
			return resp, err
		}
	}
}
