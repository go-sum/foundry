package provider

import (
	"errors"

	"github.com/go-sum/foundry/pkg/web"
)

// StubConsentRenderer is a placeholder ConsentRenderer that returns an error.
// Replace with a real implementation before enabling the OAuth provider in production.
type StubConsentRenderer struct{}

// NewStubConsentRenderer returns a ConsentRenderer that always errors.
func NewStubConsentRenderer() ConsentRenderer {
	return StubConsentRenderer{}
}

func (StubConsentRenderer) ConsentPage(_ *web.Context, _ ConsentPageData) (web.Response, error) {
	return web.Response{}, errors.New("consent UI not implemented")
}
