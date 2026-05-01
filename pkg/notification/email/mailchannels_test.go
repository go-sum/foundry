package email_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-sum/foundry/pkg/notification/email"
)

func TestNew_MailChannels_MissingAPIKey_ReturnsErrInvalidConfig(t *testing.T) {
	_, err := email.New(email.Config{
		Provider: email.ProviderMailChannels,
		APIKey:   "",
		From:     "sender@example.com",
	}, nil)
	if err == nil {
		t.Fatal("New returned nil error, want ErrInvalidConfig")
	}
	if !errors.Is(err, email.ErrInvalidConfig) {
		t.Errorf("errors.Is(err, ErrInvalidConfig) = false; err = %v", err)
	}
}

func TestNew_MailChannels_MissingFrom_ReturnsErrInvalidConfig(t *testing.T) {
	_, err := email.New(email.Config{
		Provider: email.ProviderMailChannels,
		APIKey:   "key-123",
		From:     "",
	}, nil)
	if err == nil {
		t.Fatal("New returned nil error, want ErrInvalidConfig")
	}
	if !errors.Is(err, email.ErrInvalidConfig) {
		t.Errorf("errors.Is(err, ErrInvalidConfig) = false; err = %v", err)
	}
}

func TestMailChannels_Send_Success(t *testing.T) {
	type mcAddress struct {
		Email string `json:"email"`
	}
	type mcPersonalization struct {
		To []mcAddress `json:"to"`
	}
	type mcContent struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	type payload struct {
		Personalizations []mcPersonalization `json:"personalizations"`
		From             mcAddress           `json:"from"`
		Subject          string              `json:"subject"`
		Content          []mcContent         `json:"content"`
	}

	var captured payload
	var capturedAPIKey string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAPIKey = r.Header.Get("X-Api-Key")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	s, err := email.New(email.Config{
		Provider: email.ProviderMailChannels,
		APIKey:   "mc-api-key",
		From:     "noreply@example.com",
		BaseURL:  srv.URL,
	}, nil)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	msg := email.Message{
		To:      "recipient@example.com",
		Subject: "Test Subject",
		Text:    "plain text body",
		HTML:    "<p>html body</p>",
	}
	if err := s.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	if capturedAPIKey != "mc-api-key" {
		t.Errorf("X-Api-Key = %q, want %q", capturedAPIKey, "mc-api-key")
	}
	if len(captured.Personalizations) != 1 {
		t.Fatalf("Personalizations length = %d, want 1", len(captured.Personalizations))
	}
	if len(captured.Personalizations[0].To) != 1 || captured.Personalizations[0].To[0].Email != "recipient@example.com" {
		t.Errorf("To = %v, want [{recipient@example.com}]", captured.Personalizations[0].To)
	}
	if captured.From.Email != "noreply@example.com" {
		t.Errorf("From.Email = %q, want %q", captured.From.Email, "noreply@example.com")
	}
	if captured.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", captured.Subject, "Test Subject")
	}
	if len(captured.Content) != 2 {
		t.Fatalf("Content length = %d, want 2", len(captured.Content))
	}
	if captured.Content[0].Type != "text/plain" || captured.Content[0].Value != "plain text body" {
		t.Errorf("Content[0] = %+v, want {text/plain, plain text body}", captured.Content[0])
	}
	if captured.Content[1].Type != "text/html" || captured.Content[1].Value != "<p>html body</p>" {
		t.Errorf("Content[1] = %+v, want {text/html, <p>html body</p>}", captured.Content[1])
	}
}

func TestMailChannels_Send_EmptyTo_ReturnsError(t *testing.T) {
	s, err := email.New(email.Config{
		Provider: email.ProviderMailChannels,
		APIKey:   "key",
		From:     "sender@example.com",
		BaseURL:  "https://api.mailchannels.net/tx/v1/send",
	}, nil)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	msg := email.Message{
		Subject: "No recipient",
		Text:    "body",
	}
	if err := s.Send(context.Background(), msg); err == nil {
		t.Error("Send returned nil, want error for empty To")
	}
}

func TestMailChannels_Send_5xx_ReturnsErrTransient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	t.Cleanup(srv.Close)

	s, err := email.New(email.Config{
		Provider: email.ProviderMailChannels,
		APIKey:   "key",
		From:     "sender@example.com",
		BaseURL:  srv.URL,
	}, nil)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	msg := email.Message{
		To:      "x@example.com",
		Subject: "transient test",
		Text:    "body",
	}
	err = s.Send(context.Background(), msg)
	if err == nil {
		t.Fatal("Send returned nil, want error")
	}
	if !errors.Is(err, email.ErrTransient) {
		t.Errorf("errors.Is(err, ErrTransient) = false; err = %v", err)
	}
}

func TestMailChannels_Send_4xx_NotTransient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	t.Cleanup(srv.Close)

	s, err := email.New(email.Config{
		Provider: email.ProviderMailChannels,
		APIKey:   "key",
		From:     "sender@example.com",
		BaseURL:  srv.URL,
	}, nil)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	msg := email.Message{
		To:      "x@example.com",
		Subject: "4xx test",
		Text:    "body",
	}
	err = s.Send(context.Background(), msg)
	if err == nil {
		t.Fatal("Send returned nil, want error")
	}
	if errors.Is(err, email.ErrTransient) {
		t.Errorf("errors.Is(err, ErrTransient) = true, want false for 4xx errors")
	}
}

func TestMailChannels_Send_TextOnly_ContentHasOnlyPlain(t *testing.T) {
	type mcContent struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	type payload struct {
		Content []mcContent `json:"content"`
	}

	var captured payload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	s, err := email.New(email.Config{
		Provider: email.ProviderMailChannels,
		APIKey:   "key",
		From:     "sender@example.com",
		BaseURL:  srv.URL,
	}, nil)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	msg := email.Message{
		To:      "x@example.com",
		Subject: "text only",
		Text:    "plain only",
		// HTML is empty
	}
	if err := s.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	if len(captured.Content) != 1 {
		t.Fatalf("Content length = %d, want 1", len(captured.Content))
	}
	if captured.Content[0].Type != "text/plain" {
		t.Errorf("Content[0].Type = %q, want text/plain", captured.Content[0].Type)
	}
}

func TestMailChannels_Send_HTMLOnly_ContentHasOnlyHTML(t *testing.T) {
	type mcContent struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	type payload struct {
		Content []mcContent `json:"content"`
	}

	var captured payload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	s, err := email.New(email.Config{
		Provider: email.ProviderMailChannels,
		APIKey:   "key",
		From:     "sender@example.com",
		BaseURL:  srv.URL,
	}, nil)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	msg := email.Message{
		To:      "x@example.com",
		Subject: "html only",
		HTML:    "<p>html only</p>",
		// Text is empty
	}
	if err := s.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	if len(captured.Content) != 1 {
		t.Fatalf("Content length = %d, want 1", len(captured.Content))
	}
	if captured.Content[0].Type != "text/html" {
		t.Errorf("Content[0].Type = %q, want text/html", captured.Content[0].Type)
	}
}

func TestMailChannels_Send_FromFallback(t *testing.T) {
	type mcAddress struct {
		Email string `json:"email"`
	}
	type payload struct {
		From mcAddress `json:"from"`
	}

	var captured payload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	s, err := email.New(email.Config{
		Provider: email.ProviderMailChannels,
		APIKey:   "key",
		From:     "fallback@example.com",
		BaseURL:  srv.URL,
	}, nil)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	msg := email.Message{
		To:      "x@example.com",
		Subject: "from fallback test",
		Text:    "body",
		// From is empty → should use Config.From
	}
	if err := s.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	if captured.From.Email != "fallback@example.com" {
		t.Errorf("From.Email = %q, want %q (fallback)", captured.From.Email, "fallback@example.com")
	}
}
