// Package flash renders one-time user-facing flash messages as Alert components.
package flash

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/go-sum/foundry/pkg/componentry/ui/feedback"
)

const cookieName = "flash"

// Type identifies the visual style of a flash message.
type Type string

const (
	TypeSuccess Type = "success"
	TypeInfo    Type = "info"
	TypeWarning Type = "warning"
	TypeError   Type = "error"
)

// Message is a single flash notification.
type Message struct {
	Type Type
	Text string
}

// Render maps flash messages to Alert components.
// Returns an empty text node when msgs is nil or empty.
func Render(msgs []Message) g.Node {
	if len(msgs) == 0 {
		return g.Text("")
	}
	nodes := make([]g.Node, len(msgs))
	for i, msg := range msgs {
		nodes[i] = feedback.Alert.Root(
			feedback.AlertProps{Variant: alertVariant(msg.Type), Dismissible: true},
			feedback.Alert.Description(g.Text(msg.Text)),
		)
	}
	return g.Group(nodes)
}

// RenderOOB maps flash messages to Alert components configured for out-of-band
// insertion into #flash. The hx-swap-oob="beforeend:#flash" attribute is added
// to each alert's root via Extra.
func RenderOOB(msgs []Message) g.Node {
	if len(msgs) == 0 {
		return g.Text("")
	}
	nodes := make([]g.Node, len(msgs))
	for i, msg := range msgs {
		nodes[i] = feedback.Alert.Root(
			feedback.AlertProps{
				Variant:     alertVariant(msg.Type),
				Dismissible: true,
				Extra:       []g.Node{g.Attr("hx-swap-oob", "beforeend:#flash")},
			},
			feedback.Alert.Description(g.Text(msg.Text)),
		)
	}
	return g.Group(nodes)
}

// RenderContainer renders the flash container div that holds in-page messages.
// Place this in your page template to enable Render output.
func RenderContainer(msgs []Message) g.Node {
	return h.Div(
		h.ID("flash"),
		h.Class("grid gap-2"),
		Render(msgs),
	)
}

// Set encodes msgs into a cookie-safe value on w.
func Set(w http.ResponseWriter, msgs []Message) error {
	data, err := json.Marshal(msgs)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    base64.RawURLEncoding.EncodeToString(data),
		Path:     "/",
		MaxAge:   60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// GetAll reads and clears all flash messages from the request cookie.
// Returns a non-nil empty slice when no flash cookie is present.
func GetAll(r *http.Request, w http.ResponseWriter) ([]Message, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return []Message{}, nil
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	data, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, err
	}

	var msgs []Message
	if err := json.Unmarshal(data, &msgs); err != nil {
		return nil, err
	}

	return msgs, nil
}

// Success sets a single success flash message.
func Success(w http.ResponseWriter, text string) error {
	return Set(w, []Message{{Type: TypeSuccess, Text: text}})
}

// Info sets a single info flash message.
func Info(w http.ResponseWriter, text string) error {
	return Set(w, []Message{{Type: TypeInfo, Text: text}})
}

// Warning sets a single warning flash message.
func Warning(w http.ResponseWriter, text string) error {
	return Set(w, []Message{{Type: TypeWarning, Text: text}})
}

// Error sets a single error flash message.
func Error(w http.ResponseWriter, text string) error {
	return Set(w, []Message{{Type: TypeError, Text: text}})
}

func alertVariant(t Type) feedback.AlertVariant {
	switch t {
	case TypeError:
		return feedback.AlertDestructive
	default:
		return feedback.AlertDefault
	}
}
