package web

// ErrorRenderer negotiates and renders an error response for a specific context.
// Implementations live in the application layer (e.g., starter/internal/app).
type ErrorRenderer interface {
	RenderError(c *Context, e *Error) Response
}
