package email_test

import (
	"context"
	"testing"

	"github.com/go-sum/foundry/pkg/notification/email"
)

func TestNew_Log_NoAPIKeyRequired_Succeeds(t *testing.T) {
	_, err := email.New(email.Config{
		Provider: email.ProviderLog,
		// APIKey and From intentionally omitted
	}, nil)
	if err != nil {
		t.Fatalf("New(log) returned error: %v", err)
	}
}

func TestLogSender_Send_ReturnsNil(t *testing.T) {
	s, err := email.New(email.Config{Provider: email.ProviderLog}, nil)
	if err != nil {
		t.Fatalf("New(log) returned error: %v", err)
	}

	msg := email.Message{
		To:      "recipient@example.com",
		From:    "sender@example.com",
		Subject: "Test Subject",
		Text:    "plain body",
		HTML:    "<p>html body</p>",
	}
	if err := s.Send(context.Background(), msg); err != nil {
		t.Errorf("Send returned error: %v", err)
	}
}

func TestLogSender_Send_NilLogger_UsesDefault(t *testing.T) {
	// Construct via New with nil logger — should not panic, should use slog.Default().
	s, err := email.New(email.Config{Provider: email.ProviderLog}, nil)
	if err != nil {
		t.Fatalf("New(log, nil) returned error: %v", err)
	}

	msg := email.Message{
		To:      "x@example.com",
		Subject: "nil logger test",
		Text:    "body",
	}
	if err := s.Send(context.Background(), msg); err != nil {
		t.Errorf("Send with nil logger returned error: %v", err)
	}
}
