package webhook_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-sum/notification"
	"github.com/go-sum/notification/webhook"
)

func TestNew_DefaultTimeout(t *testing.T) {
	s, err := webhook.New(webhook.Config{}, nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if s == nil {
		t.Fatal("New returned nil sender")
	}
}

func TestSender_Send_Success(t *testing.T) {
	var capturedContentType string
	var capturedBody notification.Notification

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	s, err := webhook.New(webhook.Config{DefaultURL: srv.URL}, nil)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	n := notification.Notification{
		ID:      "wh-001",
		Subject: "webhook test",
		Body:    "webhook body",
	}
	if err := s.Send(context.Background(), n); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	if capturedContentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", capturedContentType, "application/json")
	}
	if capturedBody.ID != "wh-001" {
		t.Errorf("body.ID = %q, want %q", capturedBody.ID, "wh-001")
	}
	if capturedBody.Subject != "webhook test" {
		t.Errorf("body.Subject = %q, want %q", capturedBody.Subject, "webhook test")
	}
}

func TestSender_Send_MissingURL_ReturnsError(t *testing.T) {
	s, err := webhook.New(webhook.Config{DefaultURL: ""}, nil)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	n := notification.Notification{
		Subject:  "no url",
		Metadata: map[string]string{}, // no "url" key
	}
	if err := s.Send(context.Background(), n); err == nil {
		t.Error("Send returned nil, want error for missing URL")
	}
}

func TestSender_Send_MetadataURL_OverridesDefault(t *testing.T) {
	// Set up two servers; only the override server should receive the request.
	var overrideReceived bool
	overrideSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		overrideReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(overrideSrv.Close)

	var defaultReceived bool
	defaultSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defaultReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(defaultSrv.Close)

	s, err := webhook.New(webhook.Config{DefaultURL: defaultSrv.URL}, nil)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	n := notification.Notification{
		Metadata: map[string]string{
			"url": overrideSrv.URL,
		},
	}
	if err := s.Send(context.Background(), n); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	if !overrideReceived {
		t.Error("override server did not receive the request")
	}
	if defaultReceived {
		t.Error("default server received the request but should not have")
	}
}

func TestSender_Send_CustomHeaders(t *testing.T) {
	var capturedHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get("X-Custom-Token")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	s, err := webhook.New(webhook.Config{
		DefaultURL: srv.URL,
		Headers:    map[string]string{"X-Custom-Token": "secret-token"},
	}, nil)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	if err := s.Send(context.Background(), notification.Notification{}); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	if capturedHeader != "secret-token" {
		t.Errorf("X-Custom-Token = %q, want %q", capturedHeader, "secret-token")
	}
}

func TestSender_Send_5xx_ReturnsErrTransient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("service unavailable"))
	}))
	t.Cleanup(srv.Close)

	s, err := webhook.New(webhook.Config{DefaultURL: srv.URL}, nil)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	err = s.Send(context.Background(), notification.Notification{})
	if err == nil {
		t.Fatal("Send returned nil, want error")
	}
	if !errors.Is(err, notification.ErrTransient) {
		t.Errorf("errors.Is(err, ErrTransient) = false; err = %v", err)
	}
}

func TestSender_Send_4xx_NotTransient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte("unprocessable"))
	}))
	t.Cleanup(srv.Close)

	s, err := webhook.New(webhook.Config{DefaultURL: srv.URL}, nil)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	err = s.Send(context.Background(), notification.Notification{})
	if err == nil {
		t.Fatal("Send returned nil, want error")
	}
	if errors.Is(err, notification.ErrTransient) {
		t.Errorf("errors.Is(err, ErrTransient) = true, want false for 4xx errors")
	}
}
