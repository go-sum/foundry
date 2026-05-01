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

func TestNew_Resend_MissingAPIKey_ReturnsErrInvalidConfig(t *testing.T) {
	_, err := email.New(email.Config{
		Provider: email.ProviderResend,
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

func TestNew_Resend_MissingFrom_ReturnsErrInvalidConfig(t *testing.T) {
	_, err := email.New(email.Config{
		Provider: email.ProviderResend,
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

func TestNew_Resend_HTTPBaseURL_ReturnsErrInvalidConfig(t *testing.T) {
	_, err := email.New(email.Config{
		Provider: email.ProviderResend,
		APIKey:   "key",
		From:     "sender@example.com",
		BaseURL:  "http://api.resend.com/emails",
	}, nil)
	if err == nil {
		t.Fatal("New returned nil error, want ErrInvalidConfig for http:// URL")
	}
	if !errors.Is(err, email.ErrInvalidConfig) {
		t.Errorf("errors.Is(err, ErrInvalidConfig) = false; err = %v", err)
	}
}

func TestNew_Resend_HTTPSBaseURL_Succeeds(t *testing.T) {
	_, err := email.New(email.Config{
		Provider: email.ProviderResend,
		APIKey:   "key",
		From:     "sender@example.com",
		BaseURL:  "https://api.resend.com/emails",
	}, nil)
	if err != nil {
		t.Fatalf("New returned error for https:// URL: %v", err)
	}
}

func TestResend_Send_Success(t *testing.T) {
	type payload struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
		HTML    string   `json:"html"`
	}

	var captured payload
	var capturedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	s, err := email.New(email.Config{
		Provider: email.ProviderResend,
		APIKey:   "test-api-key",
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

	if capturedAuth != "Bearer test-api-key" {
		t.Errorf("Authorization = %q, want %q", capturedAuth, "Bearer test-api-key")
	}
	if captured.From != "noreply@example.com" {
		t.Errorf("From = %q, want %q", captured.From, "noreply@example.com")
	}
	if len(captured.To) != 1 || captured.To[0] != "recipient@example.com" {
		t.Errorf("To = %v, want [recipient@example.com]", captured.To)
	}
	if captured.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", captured.Subject, "Test Subject")
	}
	if captured.Text != "plain text body" {
		t.Errorf("Text = %q, want %q", captured.Text, "plain text body")
	}
	if captured.HTML != "<p>html body</p>" {
		t.Errorf("HTML = %q, want %q", captured.HTML, "<p>html body</p>")
	}
}

func TestResend_Send_EmptyTo_ReturnsError(t *testing.T) {
	s, err := email.New(email.Config{
		Provider: email.ProviderResend,
		APIKey:   "key",
		From:     "sender@example.com",
		BaseURL:  "https://api.resend.com/emails",
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

func TestResend_Send_5xx_ReturnsErrTransient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	t.Cleanup(srv.Close)

	s, err := email.New(email.Config{
		Provider: email.ProviderResend,
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

func TestResend_Send_4xx_NotTransient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	t.Cleanup(srv.Close)

	s, err := email.New(email.Config{
		Provider: email.ProviderResend,
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

func TestResend_Send_FromFallback(t *testing.T) {
	type payload struct {
		From string `json:"from"`
	}

	var captured payload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	s, err := email.New(email.Config{
		Provider: email.ProviderResend,
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

	if captured.From != "fallback@example.com" {
		t.Errorf("From = %q, want %q (fallback)", captured.From, "fallback@example.com")
	}
}
