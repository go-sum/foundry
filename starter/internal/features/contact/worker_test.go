package contact

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-sum/foundry/pkg/notification/email"
	"github.com/go-sum/foundry/pkg/queue"
)

type fakeSender struct {
	sent []email.Message
	err  error
}

func (f *fakeSender) Send(_ context.Context, msg email.Message) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, msg)
	return nil
}

func makeJob(t *testing.T, payload any) queue.Job {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal job payload: %v", err)
	}
	return queue.Job{Payload: data}
}

func TestNotifyHandler_Success(t *testing.T) {
	sender := &fakeSender{}
	cfg := WorkerConfig{
		SendTo:   "admin@example.com",
		SendFrom: "noreply@example.com",
	}
	handler := NewNotifyHandler(sender, cfg)

	payload := NotificationPayload{
		SubmissionID: "sub-1",
		Name:         "Alice",
		Email:        "alice@example.com",
		Message:      "Hello there",
	}
	job := makeJob(t, payload)

	if err := handler(context.Background(), job); err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}

	if len(sender.sent) != 2 {
		t.Fatalf("expected 2 messages sent, got %d", len(sender.sent))
	}

	admin := sender.sent[0]
	if want := "New contact form submission from Alice"; admin.Subject != want {
		t.Errorf("admin Subject = %q, want %q", admin.Subject, want)
	}
	if admin.To != "admin@example.com" {
		t.Errorf("admin To = %q, want %q", admin.To, "admin@example.com")
	}
	if admin.From != "noreply@example.com" {
		t.Errorf("admin From = %q, want %q", admin.From, "noreply@example.com")
	}

	confirm := sender.sent[1]
	if want := "Thanks for reaching out"; confirm.Subject != want {
		t.Errorf("confirm Subject = %q, want %q", confirm.Subject, want)
	}
	if confirm.To != "alice@example.com" {
		t.Errorf("confirm To = %q, want %q", confirm.To, "alice@example.com")
	}
}

func TestNotifyHandler_InvalidPayload(t *testing.T) {
	handler := NewNotifyHandler(&fakeSender{}, WorkerConfig{})
	job := queue.Job{Payload: []byte(`not valid json`)}
	if err := handler(context.Background(), job); err == nil {
		t.Fatal("expected error for invalid payload, got nil")
	}
}

func TestNotifyHandler_SendFailure(t *testing.T) {
	sendErr := errors.New("smtp: connection refused")
	handler := NewNotifyHandler(&fakeSender{err: sendErr}, WorkerConfig{
		SendTo:   "admin@example.com",
		SendFrom: "noreply@example.com",
	})
	payload := NotificationPayload{Name: "Bob", Email: "bob@example.com", Message: "Test"}
	if err := handler(context.Background(), makeJob(t, payload)); err == nil {
		t.Fatal("expected error when send fails, got nil")
	}
}

func TestNotifyHandler_AdminBodyContent(t *testing.T) {
	sender := &fakeSender{}
	handler := NewNotifyHandler(sender, WorkerConfig{
		SendTo:   "admin@example.com",
		SendFrom: "noreply@example.com",
	})
	payload := NotificationPayload{
		Name:    "Carol",
		Email:   "carol@example.com",
		Message: "Unique message content here",
	}
	if err := handler(context.Background(), makeJob(t, payload)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sender.sent) < 1 {
		t.Fatal("expected at least one message")
	}
	if sender.sent[0].Text == "" {
		t.Error("admin message Text must not be empty")
	}
}
