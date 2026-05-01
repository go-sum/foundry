package email_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/go-sum/foundry/pkg/componentry/email"
)

type templateData struct {
	Name string
}

func TestTemplateRender(t *testing.T) {
	tmpl := email.Template[templateData]{
		Subject: func(d templateData) string {
			return "Hello, " + d.Name
		},
		HTML: func(d templateData) g.Node {
			return h.P(g.Text("Hi " + d.Name))
		},
		PlainText: func(d templateData) string {
			return email.PlainText("Hi " + d.Name)
		},
	}

	data := templateData{Name: "Alice"}
	rendered, err := tmpl.Render(data)
	if err != nil {
		t.Fatalf("Render returned unexpected error: %v", err)
	}

	wantSubject := "Hello, Alice"
	if rendered.Subject != wantSubject {
		t.Errorf("Subject = %q, want %q", rendered.Subject, wantSubject)
	}

	if rendered.HTML == "" {
		t.Error("HTML must not be empty")
	}
	if !strings.Contains(rendered.HTML, "Hi Alice") {
		t.Errorf("HTML does not contain expected content %q; got: %s", "Hi Alice", rendered.HTML)
	}

	wantText := "Hi Alice"
	if rendered.Text != wantText {
		t.Errorf("Text = %q, want %q", rendered.Text, wantText)
	}
}
