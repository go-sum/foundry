package noop_test

import (
	"context"
	"testing"

	"github.com/go-sum/foundry/pkg/notification"
	"github.com/go-sum/foundry/pkg/notification/noop"
)

func TestSender_Send_ReturnsNil(t *testing.T) {
	s := noop.New()
	n := notification.Notification{
		Subject: "test",
		Body:    "body",
	}
	if err := s.Send(context.Background(), n); err != nil {
		t.Errorf("Send returned %v, want nil", err)
	}
}
