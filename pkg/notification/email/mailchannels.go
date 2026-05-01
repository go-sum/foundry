package email

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Monitor mail log at
// https://console.mailchannels.net/logSearch
const defaultMailChannelsURL = "https://api.mailchannels.net/tx/v1/send"

type mailChannelsSender struct {
	apiKey string
	from   string
	apiURL string
	client *http.Client
}

var _ Sender = (*mailChannelsSender)(nil)

func newMailChannels(cfg Config) (*mailChannelsSender, error) {
	if cfg.From == "" {
		return nil, fmt.Errorf("%w: mailchannels: From is required", ErrInvalidConfig)
	}
	apiURL := cfg.BaseURL
	if apiURL == "" {
		apiURL = defaultMailChannelsURL
	}
	if _, err := validateHTTPConfig("mailchannels", cfg.APIKey, apiURL); err != nil {
		return nil, err
	}
	return &mailChannelsSender{
		apiKey: cfg.APIKey,
		from:   cfg.From,
		apiURL: apiURL,
		client: httpClient(cfg.Timeout, nil),
	}, nil
}

type mcPayload struct {
	Personalizations []mcPersonalization `json:"personalizations"`
	From             mcAddress           `json:"from"`
	Subject          string              `json:"subject"`
	Content          []mcContent         `json:"content"`
}

type mcPersonalization struct {
	To []mcAddress `json:"to"`
}

type mcAddress struct {
	Email string `json:"email"`
}

type mcContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (s *mailChannelsSender) Send(ctx context.Context, msg Message) error {
	if msg.To == "" {
		return fmt.Errorf("email: mailchannels: To is required")
	}
	var content []mcContent
	if msg.Text != "" {
		content = append(content, mcContent{Type: "text/plain", Value: msg.Text})
	}
	if msg.HTML != "" {
		content = append(content, mcContent{Type: "text/html", Value: msg.HTML})
	}
	if len(content) == 0 {
		return fmt.Errorf("email: mailchannels: HTML or Text is required")
	}
	p := mcPayload{
		Personalizations: []mcPersonalization{{To: []mcAddress{{Email: msg.To}}}},
		From:             mcAddress{Email: resolveFrom(msg, s.from)},
		Subject:          msg.Subject,
		Content:          content,
	}
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("email: mailchannels: encoding payload: %w", err)
	}
	return doSend(ctx, s.client, s.apiURL, map[string]string{
		"X-Api-Key": s.apiKey,
	}, body)
}
