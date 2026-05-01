package email

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Monitor mail log at
// https://resend.com/emails
const defaultResendURL = "https://api.resend.com/emails"

type resendSender struct {
	apiKey string
	from   string
	apiURL string
	client *http.Client
}

// compile-time interface check
var _ Sender = (*resendSender)(nil)

func newResend(cfg Config) (*resendSender, error) {
	if cfg.From == "" {
		return nil, fmt.Errorf("%w: resend: From is required", ErrInvalidConfig)
	}
	apiURL := cfg.BaseURL
	if apiURL == "" {
		apiURL = defaultResendURL
	}
	if _, err := validateHTTPConfig("resend", cfg.APIKey, apiURL); err != nil {
		return nil, err
	}
	return &resendSender{
		apiKey: cfg.APIKey,
		from:   cfg.From,
		apiURL: apiURL,
		client: httpClient(cfg.Timeout, nil),
	}, nil
}

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
}

func (s *resendSender) Send(ctx context.Context, msg Message) error {
	if msg.To == "" {
		return fmt.Errorf("email: resend: To is required")
	}
	p := resendPayload{
		From:    resolveFrom(msg, s.from),
		To:      []string{msg.To},
		Subject: msg.Subject,
		HTML:    msg.HTML,
		Text:    msg.Text,
	}
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("email: resend: encoding payload: %w", err)
	}
	return doSend(ctx, s.client, s.apiURL, map[string]string{
		"Authorization": "Bearer " + s.apiKey,
	}, body)
}
