package notification_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-sum/notification"
)

type stubSender struct{}

func (*stubSender) Send(_ context.Context, _ notification.Notification) error { return nil }

func TestRegistry_RegisterAndNew_ReturnsCorrectSender(t *testing.T) {
	r := notification.NewRegistry()
	r.Register("stub", func(_ map[string]string) (notification.Sender, error) {
		return &stubSender{}, nil
	})

	s, err := r.New("stub", nil)
	if err != nil {
		t.Fatalf("New returned unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("New returned nil sender")
	}
	if _, ok := s.(*stubSender); !ok {
		t.Errorf("New returned %T, want *stubSender", s)
	}
}

func TestRegistry_New_UnknownProvider_ReturnsErrProviderUnknown(t *testing.T) {
	r := notification.NewRegistry()

	_, err := r.New("nonexistent", nil)
	if err == nil {
		t.Fatal("New returned nil error, want ErrProviderUnknown")
	}
	if !errors.Is(err, notification.ErrProviderUnknown) {
		t.Errorf("errors.Is(err, ErrProviderUnknown) = false; err = %v", err)
	}
}

func TestRegistry_Register_DuplicatePanics(t *testing.T) {
	r := notification.NewRegistry()
	factory := func(_ map[string]string) (notification.Sender, error) {
		return &stubSender{}, nil
	}
	r.Register("dup", factory)

	defer func() {
		rec := recover()
		if rec == nil {
			t.Error("Register with duplicate name did not panic")
		}
	}()
	r.Register("dup", factory) // must panic
}
