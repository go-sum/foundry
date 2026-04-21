package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-sum/notification"
)

const (
	defaultResendURL = "https://api.resend.com/emails"
	defaultTimeout   = 10 * time.Second
)

// ResendConfig configures the Resend email provider.
type ResendConfig struct {
	APIKey   string
	FromAddr string
	BaseURL  string        // empty defaults to https://api.resend.com/emails
	Timeout  time.Duration // zero defaults to 10s
}

// Resend delivers email via the Resend API.
type Resend struct {
	apiKey   string
	fromAddr string
	apiURL   string
	client   *http.Client
}

// NewResend constructs a Resend sender. A nil client uses http.DefaultClient
// with the configured timeout.
func NewResend(cfg ResendConfig, client *http.Client) (*Resend, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("notification: email: resend: %w: APIKey is required", notification.ErrInvalidConfig)
	}
	if cfg.FromAddr == "" {
		return nil, fmt.Errorf("notification: email: resend: %w: FromAddr is required", notification.ErrInvalidConfig)
	}
	u := cfg.BaseURL
	if u == "" {
		u = defaultResendURL
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return &Resend{
		apiKey:   cfg.APIKey,
		fromAddr: cfg.FromAddr,
		apiURL:   u,
		client:   client,
	}, nil
}

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
}

// Send implements notification.Sender.
// Extracts "to" and "from" from n.Metadata; falls back "from" to cfg.FromAddr.
// Extracts "html" from n.Metadata for HTML body; uses n.Body for plain text.
func (r *Resend) Send(ctx context.Context, n notification.Notification) error {
	to := n.Metadata["to"]
	if to == "" {
		return fmt.Errorf("notification: email: resend: missing \"to\" in notification metadata")
	}
	from := n.Metadata["from"]
	if from == "" {
		from = r.fromAddr
	}
	p := resendPayload{
		From:    from,
		To:      []string{to},
		Subject: n.Subject,
		HTML:    n.Metadata["html"],
		Text:    n.Body,
	}
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("notification: email: resend: encoding payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("notification: email: resend: creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.apiKey)

	resp, err := r.client.Do(req)
	if err != nil {
		return errors.Join(notification.ErrTransient, fmt.Errorf("notification: email: resend: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	cause := fmt.Errorf("notification: email: resend: status %d: %s", resp.StatusCode, errBody)
	if resp.StatusCode >= 500 {
		return errors.Join(notification.ErrTransient, cause)
	}
	return cause
}
