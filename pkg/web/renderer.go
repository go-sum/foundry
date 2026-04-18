package web

import "fmt"

// ErrorRenderer negotiates and renders an error response for a specific context.
// Implementations live in the application layer (e.g., starter/internal/app).
type ErrorRenderer interface {
	RenderError(c *Context, e *Error) Response
}

// Renderer renders a named template with data.
// Implementations may use html/template, gomponents, templ, or any engine.
type Renderer interface {
	Render(c *Context, status int, name string, data any) (Response, error)
}

type rendererKey struct{}

// WithRenderer returns a Middleware that stores r in the context so handlers
// can call RenderTemplate without carrying a renderer reference explicitly.
func WithRenderer(r Renderer) Middleware {
	return func(next Handler) Handler {
		return func(c *Context) (Response, error) {
			c.Set(rendererKey{}, r)
			return next(c)
		}
	}
}

// RenderTemplate renders the named template using the Renderer stored in the
// context by WithRenderer. Returns ErrInternal if no Renderer is installed.
func RenderTemplate(c *Context, status int, name string, data any) (Response, error) {
	r, ok := Get[Renderer](c, rendererKey{})
	if !ok {
		return Response{}, ErrInternal(fmt.Errorf("web: no Renderer in context; use WithRenderer middleware"))
	}
	return r.Render(c, status, name, data)
}
