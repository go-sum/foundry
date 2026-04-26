package provider

import "github.com/go-sum/web"

// ConsentRenderer produces HTML for the OAuth consent screen.
// The host application implements this interface to control layout and styling.
type ConsentRenderer interface {
	ConsentPage(c *web.Context, data ConsentPageData) (web.Response, error)
}

// ConsentPageData carries the data needed to render the consent screen.
type ConsentPageData struct {
	// Client is the requesting OAuth client.
	Client OAuthClient
	// RequestedScopes are the scopes the client is requesting.
	RequestedScopes []string
	// Params are the original authorization request parameters, to be echoed
	// back as hidden form fields in the consent POST.
	Params AuthorizeParams
}

// AuthorizeParams holds the validated parameters from an authorization request.
// It is stored in the session between GET /oauth/authorize and POST /oauth/authorize.
type AuthorizeParams struct {
	ClientID      string   `json:"client_id"`
	RedirectURI   string   `json:"redirect_uri"`
	Scopes        []string `json:"scopes"`
	State         string   `json:"state"`
	Nonce         string   `json:"nonce"`
	CodeChallenge string   `json:"code_challenge"`
}
