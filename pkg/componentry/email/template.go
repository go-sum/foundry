package email

import (
	"bytes"
	"fmt"

	g "maragu.dev/gomponents"
)

// Rendered holds the fully resolved output from a [Template].
type Rendered struct {
	Subject string
	HTML    string
	Text    string
}

// Template co-locates email subject, HTML body, and plain-text fallback for a
// given data type D. Use package-level var declarations to define templates
// declaratively rather than building Notification structs imperatively.
type Template[D any] struct {
	Subject   func(data D) string
	HTML      func(data D) g.Node
	PlainText func(data D) string
}

// Render produces a [Rendered] from typed template data.
// Returns an error if the HTML node fails to render.
func (t Template[D]) Render(data D) (Rendered, error) {
	var buf bytes.Buffer
	if err := t.HTML(data).Render(&buf); err != nil {
		return Rendered{}, fmt.Errorf("email: render html: %w", err)
	}
	return Rendered{
		Subject: t.Subject(data),
		HTML:    buf.String(),
		Text:    t.PlainText(data),
	}, nil
}
