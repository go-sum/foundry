package webhook

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

const defaultTimeout = 10 * time.Second

// Config configures the webhook sender.
type Config struct {
	DefaultURL string
	Timeout    time.Duration     // zero defaults to 10s
	Headers    map[string]string // additional headers sent with every request
}

// Sender delivers notifications as HTTP POST JSON payloads.
type Sender struct {
	defaultURL string
	headers    map[string]string
	client     *http.Client
}

// New constructs a webhook Sender. A nil client uses http.DefaultClient with
// the configured timeout.
func New(cfg Config, client *http.Client) (*Sender, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	headers := make(map[string]string, len(cfg.Headers))
	for k, v := range cfg.Headers {
		headers[k] = v
	}
	return &Sender{
		defaultURL: cfg.DefaultURL,
		headers:    headers,
		client:     client,
	}, nil
}

// Send implements notification.Sender.
// Uses n.Metadata["url"] as the target; falls back to cfg.DefaultURL.
func (s *Sender) Send(ctx context.Context, n notification.Notification) error {
	u := n.Metadata["url"]
	if u == "" {
		u = s.defaultURL
	}
	if u == "" {
		return fmt.Errorf("notification: webhook: missing target URL")
	}
	body, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("notification: webhook: encoding payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("notification: webhook: creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return errors.Join(notification.ErrTransient, fmt.Errorf("notification: webhook: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	cause := fmt.Errorf("notification: webhook: status %d: %s", resp.StatusCode, errBody)
	if resp.StatusCode >= 500 {
		return errors.Join(notification.ErrTransient, cause)
	}
	return cause
}
