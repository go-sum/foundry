// Package view provides request-scoped presentation state and rendering helpers.
package view

import (
	"github.com/go-sum/componentry/compound"
	"github.com/go-sum/foundry/internal/view/layout"
	"github.com/go-sum/web"
	"github.com/go-sum/web/render"
	"github.com/go-sum/web/router"
	"github.com/go-sum/web/secure"
	"github.com/go-sum/web/session"

	g "maragu.dev/gomponents"
)

// Request collects request-scoped presentation state needed by pages and layout.
type Request struct {
	CurrentPath     string
	HTMX            HTMXRequest
	Routes          []router.Route
	CSRFToken       string              // CSRF token for form rendering; set via WithCSRFToken.
	RequestID       string              // Correlation ID; set via WithRequestID.
	Nonce           string              // CSP nonce; set via WithNonce.
	Flash           []string            // Flash messages; set via WithFlash.
	IsAuthenticated bool                // Auth state; set via WithIsAuthenticated.
	NavConfig       compound.NavConfig  // Declarative nav config; set via WithNavConfig.
	NavSlots        compound.NavSlots   // Dynamic nav slot content; set via WithNavSlots.
}

// HTMXRequest holds HTMX-specific request state parsed from headers.
type HTMXRequest struct {
	Enabled     bool
	Boosted     bool
	Trigger     string
	Target      string
	TriggerName string
	CurrentURL  string
}

// RequestOption configures a Request after initial construction.
type RequestOption func(*Request)

// WithCSRFToken sets the CSRF token on the view request for form rendering.
func WithCSRFToken(token string) RequestOption {
	return func(r *Request) { r.CSRFToken = token }
}

// WithRequestID sets the request correlation ID on the view request.
func WithRequestID(id string) RequestOption {
	return func(r *Request) { r.RequestID = id }
}

// WithNonce sets the CSP nonce on the view request.
func WithNonce(nonce string) RequestOption {
	return func(r *Request) { r.Nonce = nonce }
}

// WithFlash appends flash messages to the view request.
func WithFlash(messages ...string) RequestOption {
	return func(r *Request) { r.Flash = append(r.Flash, messages...) }
}

// WithIsAuthenticated sets the auth state used for nav visibility filtering.
func WithIsAuthenticated(authenticated bool) RequestOption {
	return func(r *Request) { r.IsAuthenticated = authenticated }
}

// WithNavConfig sets the declarative nav configuration rendered in the page shell.
func WithNavConfig(cfg compound.NavConfig) RequestOption {
	return func(r *Request) { r.NavConfig = cfg }
}

// WithNavSlots injects dynamic slot content (theme toggles, auth controls, etc.) into the nav.
func WithNavSlots(slots compound.NavSlots) RequestOption {
	return func(r *Request) { r.NavSlots = slots }
}

// NewRequest builds request-scoped presentation state from a web.Context.
// Fields are auto-populated from context values (RequestID, CSRFToken, Nonce,
// Flash); explicit WithX options run after and override auto-populated values.
func NewRequest(c *web.Context, routes []router.Route, opts ...RequestOption) Request {
	currentPath := ""
	headers := web.NewHeaders()
	if c != nil {
		if c.URL() != nil {
			currentPath = c.URL().Path
		}
		headers = c.Headers()
	}

	r := Request{
		CurrentPath: currentPath,
		HTMX:        NewHTMXRequest(headers),
		Routes:      routes,
	}

	if c != nil {
		r.RequestID = web.RequestID(c)
		r.CSRFToken = secure.CSRFToken(c)
		r.Nonce = secure.Nonce(c)
		if sess, ok := session.FromContext(c); ok {
			if msgs, popped, _ := session.FlashPop[[]string](sess, "flash"); popped {
				r.Flash = msgs
			}
		}
	}

	for _, opt := range opts {
		opt(&r)
	}
	return r
}

// NewHTMXRequest parses HTMX headers from a web.Headers.
func NewHTMXRequest(headers web.Headers) HTMXRequest {
	return HTMXRequest{
		Enabled:     headers.Get("HX-Request") == "true",
		Boosted:     headers.Get("HX-Boosted") == "true",
		Trigger:     headers.Get("HX-Trigger"),
		Target:      headers.Get("HX-Target"),
		TriggerName: headers.Get("HX-Trigger-Name"),
		CurrentURL:  headers.Get("HX-Current-URL"),
	}
}

// IsPartial reports whether the request should receive a fragment response.
// HTMX boosted requests get the full page (they expect a full document swap).
func (r Request) IsPartial() bool {
	return r.HTMX.Enabled && !r.HTMX.Boosted
}

// Page wraps children with the shared application layout, including the nav bar when a
// NavConfig is set on the request.
func (r Request) Page(title string, children ...g.Node) g.Node {
	var nav g.Node
	if len(r.NavConfig.Sections) > 0 || r.NavConfig.Brand.Label != "" {
		nav = compound.NavMenu(compound.NavMenuProps{
			Config:          r.NavConfig,
			Slots:           r.NavSlots,
			CurrentPath:     r.CurrentPath,
			IsAuthenticated: r.IsAuthenticated,
		})
	}
	return layout.Page(layout.Props{
		Title:     title,
		Nonce:     r.Nonce,
		CSRFToken: r.CSRFToken,
		Flash:     r.Flash,
		Nav:       nav,
		Children:  children,
	})
}

// Render chooses the correct response mode. HTMX partial requests receive
// partial; all others receive full. If partial is nil, full is used.
func Render(req Request, full, partial g.Node) (web.Response, error) {
	if partial != nil && req.IsPartial() {
		return render.Fragment(partial)
	}
	return render.Component(full)
}

// RenderWithStatus is Render with an explicit HTTP status code.
func RenderWithStatus(req Request, status int, full, partial g.Node) web.Response {
	var resp web.Response
	var err error
	if partial != nil && req.IsPartial() {
		resp, err = render.FragmentWithStatus(status, partial)
	} else {
		resp, err = render.ComponentWithStatus(status, full)
	}
	if err != nil {
		// Rendering failures become a plain 500 text response. The boundary
		// will not be involved here because RenderWithStatus is called from
		// the error renderer itself; avoid infinite recursion.
		return web.Text(500, "internal server error")
	}
	return resp
}
