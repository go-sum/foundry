package email

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

var (
	ErrInvalidConfig = errors.New("email: invalid configuration")
	ErrTransient     = errors.New("email: transient failure")
)

// Message is the email to deliver.
type Message struct {
	To      string // recipient address
	From    string // overrides Config.From if non-empty
	Subject string
	HTML    string
	Text    string
}

// Sender delivers an email.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// New constructs a Sender from cfg. logger is used only by the "log" provider;
// it may be nil (falls back to slog.Default).
func New(cfg Config, logger *slog.Logger) (Sender, error) {
	switch cfg.Provider {
	case ProviderResend:
		return newResend(cfg)
	case ProviderMailChannels:
		return newMailChannels(cfg)
	case ProviderLog:
		return newLogSender(logger), nil
	default:
		return nil, fmt.Errorf("%w: unknown provider %q (want resend, mailchannels, or log)", ErrInvalidConfig, cfg.Provider)
	}
}

// validateHTTPConfig checks fields shared by all HTTP-based providers.
func validateHTTPConfig(provider, apiKey, rawURL string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("%w: %s: APIKey is required", ErrInvalidConfig, provider)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("%w: %s: BaseURL must be a valid URL", ErrInvalidConfig, provider)
	}
	host := parsed.Hostname()
	isLoopback := host == "127.0.0.1" || host == "::1" || host == "localhost"
	if parsed.Scheme != "https" && !isLoopback {
		return "", fmt.Errorf("%w: %s: BaseURL must use https scheme", ErrInvalidConfig, provider)
	}
	return rawURL, nil
}

// resolveFrom returns msg.From if non-empty, otherwise defaultFrom.
func resolveFrom(msg Message, defaultFrom string) string {
	if msg.From != "" {
		return msg.From
	}
	return defaultFrom
}

// httpClient builds a *http.Client with a timeout, or returns the provided one.
func httpClient(timeout time.Duration, override *http.Client) *http.Client {
	if override != nil {
		return override
	}
	t := timeout
	if t <= 0 {
		t = defaultTimeout
	}
	return &http.Client{Timeout: t}
}

// doSend POSTs body to url with the given headers and classifies the response.
func doSend(ctx context.Context, client *http.Client, apiURL string, headers map[string]string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("email: creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransient, err)
	}
	defer resp.Body.Close() //nolint:errcheck // response body already consumed; Close error is non-actionable
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 128))
	cause := fmt.Errorf("email: non-success status %d (body: %.128s)", resp.StatusCode, errBody)
	if resp.StatusCode >= 500 {
		return fmt.Errorf("%w: %w", ErrTransient, cause)
	}
	return cause
}
