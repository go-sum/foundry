package email_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-sum/foundry/pkg/notification"
	"github.com/go-sum/foundry/pkg/notification/email"
)

func TestNewResend_MissingAPIKey_ReturnsErrInvalidConfig(t *testing.T) {
	_, err := email.NewResend(email.ResendConfig{
		APIKey:   "",
		FromAddr: "sender@example.com",
	}, nil)
	if err == nil {
		t.Fatal("NewResend returned nil error, want ErrInvalidConfig")
	}
	if !errors.Is(err, notification.ErrInvalidConfig) {
		t.Errorf("errors.Is(err, ErrInvalidConfig) = false; err = %v", err)
	}
}

func TestNewResend_MissingFromAddr_ReturnsErrInvalidConfig(t *testing.T) {
	_, err := email.NewResend(email.ResendConfig{
		APIKey:   "key-123",
		FromAddr: "",
	}, nil)
	if err == nil {
		t.Fatal("NewResend returned nil error, want ErrInvalidConfig")
	}
	if !errors.Is(err, notification.ErrInvalidConfig) {
		t.Errorf("errors.Is(err, ErrInvalidConfig) = false; err = %v", err)
	}
}

func TestResend_Send_Success(t *testing.T) {
	type payload struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
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

	r, err := email.NewResend(email.ResendConfig{
		APIKey:   "test-api-key",
		FromAddr: "noreply@example.com",
		BaseURL:  srv.URL,
	}, nil)
	if err != nil {
		t.Fatalf("NewResend error: %v", err)
	}

	n := notification.Notification{
		Subject: "Test Subject",
		Body:    "plain text body",
		Metadata: map[string]string{
			"to": "recipient@example.com",
		},
	}
	if err := r.Send(context.Background(), n); err != nil {
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
}

func TestResend_Send_MissingTo_ReturnsError(t *testing.T) {
	r, err := email.NewResend(email.ResendConfig{
		APIKey:   "key",
		FromAddr: "sender@example.com",
	}, nil)
	if err != nil {
		t.Fatalf("NewResend error: %v", err)
	}

	n := notification.Notification{
		Subject:  "No recipient",
		Metadata: map[string]string{}, // no "to"
	}
	if err := r.Send(context.Background(), n); err == nil {
		t.Error("Send returned nil, want error for missing 'to'")
	}
}

func TestResend_Send_5xx_ReturnsErrTransient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	t.Cleanup(srv.Close)

	r, err := email.NewResend(email.ResendConfig{
		APIKey:   "key",
		FromAddr: "sender@example.com",
		BaseURL:  srv.URL,
	}, nil)
	if err != nil {
		t.Fatalf("NewResend error: %v", err)
	}

	n := notification.Notification{
		Subject:  "transient test",
		Metadata: map[string]string{"to": "x@example.com"},
	}
	err = r.Send(context.Background(), n)
	if err == nil {
		t.Fatal("Send returned nil, want error")
	}
	if !errors.Is(err, notification.ErrTransient) {
		t.Errorf("errors.Is(err, ErrTransient) = false; err = %v", err)
	}
}

func TestResend_Send_4xx_NotTransient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	t.Cleanup(srv.Close)

	r, err := email.NewResend(email.ResendConfig{
		APIKey:   "key",
		FromAddr: "sender@example.com",
		BaseURL:  srv.URL,
	}, nil)
	if err != nil {
		t.Fatalf("NewResend error: %v", err)
	}

	n := notification.Notification{
		Subject:  "4xx test",
		Metadata: map[string]string{"to": "x@example.com"},
	}
	err = r.Send(context.Background(), n)
	if err == nil {
		t.Fatal("Send returned nil, want error")
	}
	if errors.Is(err, notification.ErrTransient) {
		t.Errorf("errors.Is(err, ErrTransient) = true, want false for 4xx errors")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error message = %q, expected it to contain status code 400", err.Error())
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

	r, err := email.NewResend(email.ResendConfig{
		APIKey:   "key",
		FromAddr: "fallback@example.com",
		BaseURL:  srv.URL,
	}, nil)
	if err != nil {
		t.Fatalf("NewResend error: %v", err)
	}

	n := notification.Notification{
		Subject: "from fallback test",
		Metadata: map[string]string{
			"to":   "x@example.com",
			"from": "", // empty → should fall back to cfg.FromAddr
		},
	}
	if err := r.Send(context.Background(), n); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	if captured.From != "fallback@example.com" {
		t.Errorf("From = %q, want %q (fallback)", captured.From, "fallback@example.com")
	}
}
