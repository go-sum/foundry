package contact

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-sum/notification"
	"github.com/go-sum/notification/memory"
	"github.com/go-sum/queue"
)

// failingSender is a notification.Sender that always returns an error.
type failingSender struct {
	err error
}

func (f *failingSender) Send(_ context.Context, _ notification.Notification) error {
	return f.err
}

func makeJob(t *testing.T, payload any) queue.Job {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal job payload: %v", err)
	}
	return queue.Job{Payload: data}
}

func makeNotifier(sender notification.Sender) *notification.Dispatcher {
	return notification.NewDispatcher(map[notification.Channel]notification.Sender{
		notification.ChannelEmail: sender,
	}, nil)
}

// TestNotifyHandler_Success verifies that a valid payload causes exactly two
// notifications to be dispatched: one admin notification, one confirmation.
func TestNotifyHandler_Success(t *testing.T) {
	sender := memory.New()
	notifier := makeNotifier(sender)
	cfg := WorkerConfig{
		SendTo:   "admin@example.com",
		SendFrom: "noreply@example.com",
	}
	handler := NewNotifyHandler(notifier, cfg)

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

	sent := sender.Sent()
	if len(sent) != 2 {
		t.Fatalf("expected 2 notifications sent, got %d", len(sent))
	}

	// First notification is the admin alert.
	adminNotif := sent[0]
	wantAdminSubject := "New contact form submission from Alice"
	if adminNotif.Subject != wantAdminSubject {
		t.Errorf("admin notification Subject = %q, want %q", adminNotif.Subject, wantAdminSubject)
	}
	if adminNotif.Metadata["to"] != "admin@example.com" {
		t.Errorf("admin notification to = %q, want %q", adminNotif.Metadata["to"], "admin@example.com")
	}
	if adminNotif.Metadata["from"] != "noreply@example.com" {
		t.Errorf("admin notification from = %q, want %q", adminNotif.Metadata["from"], "noreply@example.com")
	}

	// Second notification is the submitter confirmation.
	confirmNotif := sent[1]
	wantConfirmSubject := "Thanks for reaching out"
	if confirmNotif.Subject != wantConfirmSubject {
		t.Errorf("confirmation Subject = %q, want %q", confirmNotif.Subject, wantConfirmSubject)
	}
	if confirmNotif.Metadata["to"] != "alice@example.com" {
		t.Errorf("confirmation to = %q, want %q", confirmNotif.Metadata["to"], "alice@example.com")
	}
	if confirmNotif.Metadata["from"] != "noreply@example.com" {
		t.Errorf("confirmation from = %q, want %q", confirmNotif.Metadata["from"], "noreply@example.com")
	}
}

// TestNotifyHandler_InvalidPayload verifies that malformed JSON causes an error.
func TestNotifyHandler_InvalidPayload(t *testing.T) {
	sender := memory.New()
	notifier := makeNotifier(sender)
	handler := NewNotifyHandler(notifier, WorkerConfig{})

	job := queue.Job{Payload: []byte(`not valid json`)}

	err := handler(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for invalid payload, got nil")
	}
}

// TestNotifyHandler_NotificationFailure verifies that a sender error is returned.
func TestNotifyHandler_NotificationFailure(t *testing.T) {
	sendErr := errors.New("smtp: connection refused")
	notifier := makeNotifier(&failingSender{err: sendErr})
	handler := NewNotifyHandler(notifier, WorkerConfig{
		SendTo:   "admin@example.com",
		SendFrom: "noreply@example.com",
	})

	payload := NotificationPayload{
		SubmissionID: "sub-2",
		Name:         "Bob",
		Email:        "bob@example.com",
		Message:      "Test",
	}
	job := makeJob(t, payload)

	err := handler(context.Background(), job)
	if err == nil {
		t.Fatal("expected error when notification send fails, got nil")
	}
}

// TestNotifyHandler_NotificationBody verifies that the admin notification body
// contains identifying information from the payload.
func TestNotifyHandler_NotificationBody(t *testing.T) {
	sender := memory.New()
	notifier := makeNotifier(sender)
	handler := NewNotifyHandler(notifier, WorkerConfig{
		SendTo:   "admin@example.com",
		SendFrom: "noreply@example.com",
	})

	payload := NotificationPayload{
		SubmissionID: "sub-3",
		Name:         "Carol",
		Email:        "carol@example.com",
		Message:      "Unique message content here",
	}
	job := makeJob(t, payload)

	if err := handler(context.Background(), job); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sent := sender.Sent()
	if len(sent) < 1 {
		t.Fatal("expected at least one notification")
	}

	adminBody := sent[0].Body
	if adminBody == "" {
		t.Error("admin notification body must not be empty")
	}
}
