package secure

import (
	"strings"

	"github.com/go-sum/foundry/pkg/web"
)

// ClearSiteData returns middleware that emits a Clear-Site-Data response header
// containing the specified directives for every response.
//
// Supported directives: "cache", "cookies", "storage", "executionContexts".
// Pass no arguments to clear cookies and storage (the most common use case).
//
// Example:
//
//	secure.ClearSiteData()                          // clears "cookies", "storage"
//	secure.ClearSiteData("cookies", "storage", "cache")
func ClearSiteData(directives ...string) web.Middleware {
	if len(directives) == 0 {
		directives = []string{"cookies", "storage"}
	}
	// Build the header value once: "cookies", "storage"
	quoted := make([]string, len(directives))
	for i, d := range directives {
		quoted[i] = `"` + d + `"`
	}
	value := strings.Join(quoted, ", ")

	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			resp, err := next(c)
			resp.Headers.Set("Clear-Site-Data", value)
			return resp, err
		}
	}
}

// SetClearSiteData adds the Clear-Site-Data header to resp with the given directives.
// Convenience function for use inside handlers without a middleware chain.
func SetClearSiteData(resp *web.Response, directives ...string) {
	if len(directives) == 0 {
		directives = []string{"cookies", "storage"}
	}
	quoted := make([]string, len(directives))
	for i, d := range directives {
		quoted[i] = `"` + d + `"`
	}
	resp.Headers.Set("Clear-Site-Data", strings.Join(quoted, ", "))
}
