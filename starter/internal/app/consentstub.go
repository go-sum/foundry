package app

import (
	"errors"

	"github.com/go-sum/foundry/pkg/auth/provider"
	"github.com/go-sum/foundry/pkg/web"
)

// stubConsentRenderer is a placeholder ConsentRenderer that returns an error response.
// Replace this with a real providerui implementation before enabling the OAuth provider
// in production.
type stubConsentRenderer struct{}

func (stubConsentRenderer) ConsentPage(_ *web.Context, _ provider.ConsentPageData) (web.Response, error) {
	return web.Response{}, errors.New("consent UI not implemented")
}
