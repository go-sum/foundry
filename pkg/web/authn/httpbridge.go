package authn

import (
	"net/http"

	"github.com/go-sum/foundry/pkg/web"
)

// toHTTPRequest reconstructs a *http.Request from a web.Context.
// This is required because the go-webauthn library's FinishRegistration and
// FinishDiscoverableLogin methods parse the credential response from an
// *http.Request body.
func toHTTPRequest(c *web.Context) *http.Request {
	r := &http.Request{
		Method: c.Method(),
		URL:    c.URL(),
		Header: make(http.Header),
		Body:   c.Request.Body,
	}
	c.Headers().ForEach(func(name string, values []string) {
		for _, v := range values {
			r.Header.Add(name, v)
		}
	})
	return r
}
