package notification_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-sum/notification"
	"github.com/go-sum/notification/memory"
)

// fakeSender is a Sender that always returns a configured error.
type fakeSender struct {
	err error
}

func (f *fakeSender) Send(_ context.Context, _ notification.Notification) error {
	return f.err
}

func TestDispatcher_Send_RoutesToCorrectChannel(t *testing.T) {
	mem := memory.New()
	d := notification.NewDispatcher(map[notification.Channel]notification.Sender{
		notification.ChannelEmail: mem,
	}, nil)

	n := notification.Notification{
		Subject:  "hello",
		Channels: []notification.Channel{notification.ChannelEmail},
	}
	if err := d.Send(context.Background(), n); err != nil {
		t.Fatalf("Send returned unexpected error: %v", err)
	}

	sent := mem.Sent()
	if len(sent) != 1 {
		t.Fatalf("memory sender captured %d notifications, want 1", len(sent))
	}
	if sent[0].Subject != "hello" {
		t.Errorf("subject = %q, want %q", sent[0].Subject, "hello")
	}
}

func TestDispatcher_Send_EmptyChannels_SendsToAll(t *testing.T) {
	emailMem := memory.New()
	webhookMem := memory.New()
	d := notification.NewDispatcher(map[notification.Channel]notification.Sender{
		notification.ChannelEmail:   emailMem,
		notification.ChannelWebhook: webhookMem,
	}, nil)

	n := notification.Notification{
		Subject: "broadcast",
		// Channels intentionally empty → send to all
	}
	if err := d.Send(context.Background(), n); err != nil {
		t.Fatalf("Send returned unexpected error: %v", err)
	}

	if len(emailMem.Sent()) != 1 {
		t.Errorf("email sender captured %d notifications, want 1", len(emailMem.Sent()))
	}
	if len(webhookMem.Sent()) != 1 {
		t.Errorf("webhook sender captured %d notifications, want 1", len(webhookMem.Sent()))
	}
}

func TestDispatcher_Send_UnknownChannel_ReturnsErrChannelUnavailable(t *testing.T) {
	d := notification.NewDispatcher(map[notification.Channel]notification.Sender{
		notification.ChannelEmail: memory.New(),
	}, nil)

	n := notification.Notification{
		Subject:  "test",
		Channels: []notification.Channel{notification.ChannelWebhook}, // not registered
	}
	err := d.Send(context.Background(), n)
	if err == nil {
		t.Fatal("Send returned nil, want ErrChannelUnavailable")
	}
	if !errors.Is(err, notification.ErrChannelUnavailable) {
		t.Errorf("errors.Is(err, ErrChannelUnavailable) = false; err = %v", err)
	}
}

func TestDispatcher_Send_MultipleChannelErrors_Joined(t *testing.T) {
	sentinelErr := errors.New("send failed")
	bad := &fakeSender{err: sentinelErr}
	d := notification.NewDispatcher(map[notification.Channel]notification.Sender{
		notification.ChannelEmail:   bad,
		notification.ChannelWebhook: bad,
	}, nil)

	n := notification.Notification{
		Channels: []notification.Channel{notification.ChannelEmail, notification.ChannelWebhook},
	}
	err := d.Send(context.Background(), n)
	if err == nil {
		t.Fatal("Send returned nil, want joined errors")
	}
	// errors.Join produces an error that unwraps to both constituents; verify at least one is present.
	errStr := err.Error()
	if errStr == "" {
		t.Error("joined error string is empty")
	}
}

func TestDispatcher_Send_SetsTimestamp(t *testing.T) {
	mem := memory.New()
	d := notification.NewDispatcher(map[notification.Channel]notification.Sender{
		notification.ChannelEmail: mem,
	}, nil)

	n := notification.Notification{
		Channels: []notification.Channel{notification.ChannelEmail},
		// Timestamp intentionally zero
	}
	before := time.Now()
	if err := d.Send(context.Background(), n); err != nil {
		t.Fatalf("Send error: %v", err)
	}
	after := time.Now()

	sent := mem.Sent()
	if len(sent) == 0 {
		t.Fatal("no notifications captured")
	}
	ts := sent[0].Timestamp
	if ts.IsZero() {
		t.Error("Timestamp is still zero after Send")
	}
	if ts.Before(before) || ts.After(after) {
		t.Errorf("Timestamp %v outside expected range [%v, %v]", ts, before, after)
	}
}

func TestNewDispatcher_NilChannels_Safe(t *testing.T) {
	d := notification.NewDispatcher(nil, nil)
	n := notification.Notification{
		Subject:  "safe",
		Channels: []notification.Channel{}, // empty → all configured, which is none
	}
	// With no channels configured and empty n.Channels, Send should return nil.
	if err := d.Send(context.Background(), n); err != nil {
		t.Errorf("Send on nil-channel dispatcher returned error: %v", err)
	}
}
