package static

import (
	"github.com/go-sum/foundry/pkg/web"
)

const (
	immutableCacheControl = "public, max-age=31536000, immutable"
	noCacheControl        = "no-cache"
)

// VersionedCacheControl wraps handler with a two-tier cache strategy:
// requests with the version query parameter get an immutable 1-year cache,
// all others get no-cache to ensure revalidation.
func VersionedCacheControl(versionParam string) func(web.Handler) web.Handler {
	return func(next web.Handler) web.Handler {
		return func(c *web.Context) (web.Response, error) {
			resp, err := next(c)
			if err != nil {
				return resp, err
			}
			if resp.Headers.Get("Cache-Control") == "" {
				cc := noCacheControl
				if c.Request.URL.Query().Get(versionParam) != "" {
					cc = immutableCacheControl
				}
				resp.Headers.Set("Cache-Control", cc)
			}
			return resp, nil
		}
	}
}
