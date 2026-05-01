package contact

import (
	"context"
	"encoding/json"
	"fmt"

	compemail "github.com/go-sum/foundry/pkg/componentry/email"
	"github.com/go-sum/foundry/pkg/notification/email"
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

var adminEmailTemplate = compemail.Template[NotificationPayload]{
	Subject: func(p NotificationPayload) string {
		return "New contact form submission from " + p.Name
	},
	HTML: func(p NotificationPayload) g.Node {
		return compemail.Layout(compemail.LayoutProps{Title: "New Contact Submission"}, g.Group([]g.Node{
			compemail.H1("New contact form submission"),
			compemail.P("Name: " + p.Name),
			compemail.P("Email: " + p.Email),
			compemail.P("Message:"),
			compemail.P(p.Message),
		}))
	},
	PlainText: func(p NotificationPayload) string {
		return compemail.PlainText(
			"New contact form submission",
			"",
			"Name: "+p.Name,
			"Email: "+p.Email,
			"",
			"Message:",
			p.Message,
		)
	},
}

var confirmEmailTemplate = compemail.Template[NotificationPayload]{
	Subject: func(_ NotificationPayload) string {
		return "Thanks for reaching out"
	},
	HTML: func(p NotificationPayload) g.Node {
		return compemail.Layout(compemail.LayoutProps{Title: "Thanks for reaching out"}, g.Group([]g.Node{
			compemail.H1("Thanks for reaching out, " + p.Name + "!"),
			compemail.P("We've received your message and will get back to you soon."),
			compemail.P("Your message:"),
			compemail.P(p.Message),
		}))
	},
	PlainText: func(p NotificationPayload) string {
		return compemail.PlainText(
			"Thanks for reaching out, "+p.Name+"!",
			"",
			"We've received your message and will get back to you soon.",
			"",
			"Your message:",
			p.Message,
		)
	},
}

// NewNotifyHandler returns a queue.HandlerFunc that dispatches email notifications
// for a submitted contact form.
func NewNotifyHandler(sender email.Sender, cfg WorkerConfig) queue.HandlerFunc {
	return func(ctx context.Context, job queue.Job) error {
		var payload NotificationPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("contact: unmarshal payload: %w", err)
		}

		adminRendered, err := adminEmailTemplate.Render(payload)
		if err != nil {
			return fmt.Errorf("contact: render admin email: %w", err)
		}
		if err := sender.Send(ctx, email.Message{
			To:      cfg.SendTo,
			From:    cfg.SendFrom,
			Subject: adminRendered.Subject,
			HTML:    adminRendered.HTML,
			Text:    adminRendered.Text,
		}); err != nil {
			return fmt.Errorf("contact: send admin notification: %w", err)
		}

		confirmRendered, err := confirmEmailTemplate.Render(payload)
		if err != nil {
			return fmt.Errorf("contact: render confirmation email: %w", err)
		}
		if err := sender.Send(ctx, email.Message{
			To:      payload.Email,
			From:    cfg.SendFrom,
			Subject: confirmRendered.Subject,
			HTML:    confirmRendered.HTML,
			Text:    confirmRendered.Text,
		}); err != nil {
			return fmt.Errorf("contact: send confirmation: %w", err)
		}

		return nil
	}
}
