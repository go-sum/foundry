package memory_test

import (
	"context"
	"testing"

	"github.com/go-sum/foundry/pkg/notification"
	"github.com/go-sum/foundry/pkg/notification/memory"
)

func TestSender_Send_CapturesNotifications(t *testing.T) {
	s := memory.New()

	n1 := notification.Notification{Subject: "first", Body: "body1"}
	n2 := notification.Notification{Subject: "second", Body: "body2"}

	if err := s.Send(context.Background(), n1); err != nil {
		t.Fatalf("Send(n1) returned error: %v", err)
	}
	if err := s.Send(context.Background(), n2); err != nil {
		t.Fatalf("Send(n2) returned error: %v", err)
	}

	sent := s.Sent()
	if len(sent) != 2 {
		t.Fatalf("Sent() returned %d items, want 2", len(sent))
	}
	if sent[0].Subject != "first" {
		t.Errorf("sent[0].Subject = %q, want %q", sent[0].Subject, "first")
	}
	if sent[1].Subject != "second" {
		t.Errorf("sent[1].Subject = %q, want %q", sent[1].Subject, "second")
	}
}

func TestSender_Sent_ReturnsCopy(t *testing.T) {
	s := memory.New()
	n := notification.Notification{Subject: "original"}
	if err := s.Send(context.Background(), n); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	copy1 := s.Sent()
	// Mutate the returned slice — should not affect internal state.
	copy1[0].Subject = "mutated"

	copy2 := s.Sent()
	if copy2[0].Subject != "original" {
		t.Errorf("internal state was mutated: subject = %q, want %q", copy2[0].Subject, "original")
	}
}

func TestSender_Reset_ClearsCapture(t *testing.T) {
	s := memory.New()
	n := notification.Notification{Subject: "to be cleared"}
	if err := s.Send(context.Background(), n); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	if len(s.Sent()) == 0 {
		t.Fatal("expected one notification before Reset")
	}

	s.Reset()

	if got := len(s.Sent()); got != 0 {
		t.Errorf("after Reset: Sent() returned %d items, want 0", got)
	}
}
