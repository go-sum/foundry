package contact

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-sum/foundry/pkg/componentry/email"
	"github.com/go-sum/foundry/pkg/notification"
	"github.com/go-sum/foundry/pkg/queue"
	g "maragu.dev/gomponents"
)

// QueueName is the queue used for contact notification jobs.
const QueueName = "contact.notify"

// WorkerConfig holds addresses for outbound email.
type WorkerConfig struct {
	SendTo   string
	SendFrom string
}

// NewNotifyHandler returns a queue.HandlerFunc that dispatches email notifications
// for a submitted contact form.
func NewNotifyHandler(notifier *notification.Dispatcher, cfg WorkerConfig) queue.HandlerFunc {
	return func(ctx context.Context, job queue.Job) error {
		var payload NotificationPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("contact: unmarshal payload: %w", err)
		}

		adminHTML, err := renderHTML(notificationBody(payload))
		if err != nil {
			return fmt.Errorf("contact: render admin html: %w", err)
		}
		adminNotif := notification.Notification{
			Subject:  "New contact form submission from " + payload.Name,
			Body:     notificationText(payload),
			Channels: []notification.Channel{notification.ChannelEmail},
			Metadata: map[string]string{
				"to":   cfg.SendTo,
				"from": cfg.SendFrom,
				"html": adminHTML,
			},
		}
		if err := notifier.Send(ctx, adminNotif); err != nil {
			return fmt.Errorf("contact: send admin notification: %w", err)
		}

		confirmHTML, err := renderHTML(confirmationBody(payload))
		if err != nil {
			return fmt.Errorf("contact: render confirmation html: %w", err)
		}
		confirmNotif := notification.Notification{
			Subject:  "Thanks for reaching out",
			Body:     confirmationText(payload),
			Channels: []notification.Channel{notification.ChannelEmail},
			Metadata: map[string]string{
				"to":   payload.Email,
				"from": cfg.SendFrom,
				"html": confirmHTML,
			},
		}
		if err := notifier.Send(ctx, confirmNotif); err != nil {
			return fmt.Errorf("contact: send confirmation: %w", err)
		}

		return nil
	}
}

func renderHTML(node g.Node) (string, error) {
	var buf bytes.Buffer
	if err := node.Render(&buf); err != nil {
		return "", fmt.Errorf("contact: render html: %w", err)
	}
	return buf.String(), nil
}

func notificationBody(p NotificationPayload) g.Node {
	return email.Layout(email.LayoutProps{Title: "New Contact Submission"}, g.Group([]g.Node{
		email.H1("New contact form submission"),
		email.P("Name: " + p.Name),
		email.P("Email: " + p.Email),
		email.P("Message:"),
		email.P(p.Message),
	}))
}

func notificationText(p NotificationPayload) string {
	return email.PlainText(
		"New contact form submission",
		"",
		"Name: "+p.Name,
		"Email: "+p.Email,
		"",
		"Message:",
		p.Message,
	)
}

func confirmationBody(p NotificationPayload) g.Node {
	return email.Layout(email.LayoutProps{Title: "Thanks for reaching out"}, g.Group([]g.Node{
		email.H1("Thanks for reaching out, " + p.Name + "!"),
		email.P("We've received your message and will get back to you soon."),
		email.P("Your message:"),
		email.P(p.Message),
	}))
}

func confirmationText(p NotificationPayload) string {
	return email.PlainText(
		"Thanks for reaching out, "+p.Name+"!",
		"",
		"We've received your message and will get back to you soon.",
		"",
		"Your message:",
		p.Message,
	)
}
