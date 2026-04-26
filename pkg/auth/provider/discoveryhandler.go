package provider

import (
	"github.com/go-sum/web"
	"github.com/go-sum/web/router"
)

// DiscoveryHandler serves the OIDC discovery document.
type DiscoveryHandler struct {
	config Config
	router *router.Router
}

type discoveryDocument struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint,omitempty"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

// Serve handles GET /.well-known/openid-configuration.
func (h *DiscoveryHandler) Serve(c *web.Context) (web.Response, error) {
	res := router.NewResolver(h.router)
	authzPath := res.Path(RouteAuthorize)()
	tokenPath := res.Path(RouteToken)()
	userinfoPath := res.Path(RouteUserinfo)()

	doc := discoveryDocument{
		Issuer:                            h.config.Issuer,
		AuthorizationEndpoint:             h.config.Issuer + authzPath,
		TokenEndpoint:                     h.config.Issuer + tokenPath,
		UserinfoEndpoint:                  h.config.Issuer + userinfoPath,
		ResponseTypesSupported:            []string{"code"},
		CodeChallengeMethodsSupported:     []string{"S256"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_post", "none"},
	}
	return web.JSON(200, doc), nil
}
