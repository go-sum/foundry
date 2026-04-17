package adapt

import "net/http"

// WriteEarlyHints sends a 103 Early Hints informational response to the
// client, advising it to preload the given Link header values before the
// main response arrives. This enables browsers to fetch critical
// subresources (stylesheets, scripts, fonts) in parallel with server-side
// processing.
//
// links is a slice of RFC 8288 Link header values, e.g.:
//
//	"</style.css>; rel=preload; as=style"
//	"</app.js>; rel=preload; as=script"
//
// This is a best-effort operation. On transports that do not support
// 1xx informational responses (standard HTTP/1.1 net/http servers)
// this call is a no-op. It works on servers whose ResponseWriter
// implements the optional informational-response interface below.
//
// Usage — call before h(c):
//
//	func myHandler(c *web.Context) web.Response {
//	    adapt.WriteEarlyHints(w, []string{
//	        `</static/app.css>; rel=preload; as=style`,
//	        `</static/app.js>; rel=preload; as=script`,
//	    })
//	    // ... expensive DB query ...
//	    return web.HTML(200, page)
//	}
//
// Note: the handler does not have access to the underlying http.ResponseWriter.
// WriteEarlyHints is intended for use in middleware or in a thin wrapper
// around the adapt.ToHTTPHandler call site where w is available.
func WriteEarlyHints(w http.ResponseWriter, links []string) {
	if len(links) == 0 {
		return
	}
	// Early Hints requires the ResponseWriter to support 1xx informational
	// responses. This is exposed via an optional interface. Standard
	// net/http ResponseWriters do not implement it; HTTP/2 and HTTP/3
	// server implementations may.
	type informationalResponder interface {
		WriteInfoHeader(statusCode int, header http.Header) error
	}
	h := make(http.Header, 1)
	for _, link := range links {
		h.Add("Link", link)
	}
	if iw, ok := w.(informationalResponder); ok {
		_ = iw.WriteInfoHeader(http.StatusEarlyHints, h)
	}
}
