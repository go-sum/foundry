package config

import (
	"errors"
	"strings"
	"testing"
)

func TestCloser_EmptyClose(t *testing.T) {
	var c Closer
	if err := c.Close(); err != nil {
		t.Errorf("Close() on empty Closer = %v, want nil", err)
	}
}

func TestCloser_SingleSuccess(t *testing.T) {
	var c Closer
	called := false
	c.Add("svc", func() error {
		called = true
		return nil
	})
	if err := c.Close(); err != nil {
		t.Errorf("Close() = %v, want nil", err)
	}
	if !called {
		t.Error("cleanup function was not called")
	}
}

func TestCloser_LIFOOrdering(t *testing.T) {
	var c Closer
	var order []string

	c.Add("first", func() error {
		order = append(order, "first")
		return nil
	})
	c.Add("second", func() error {
		order = append(order, "second")
		return nil
	})

	if err := c.Close(); err != nil {
		t.Fatalf("Close() = %v, want nil", err)
	}

	if len(order) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(order))
	}
	if order[0] != "second" || order[1] != "first" {
		t.Errorf("LIFO order = %v, want [second first]", order)
	}
}

func TestCloser_ErrorAggregation(t *testing.T) {
	var c Closer
	err1 := errors.New("failure one")
	err2 := errors.New("failure two")

	c.Add("alpha", func() error { return err1 })
	c.Add("beta", func() error { return err2 })

	err := c.Close()
	if err == nil {
		t.Fatal("Close() = nil, want non-nil error")
	}
	if !errors.Is(err, err1) {
		t.Errorf("errors.Is(err, err1) = false, want true")
	}
	if !errors.Is(err, err2) {
		t.Errorf("errors.Is(err, err2) = false, want true")
	}
}

func TestCloser_PartialFailure(t *testing.T) {
	var c Closer
	called := make([]string, 0, 2)
	errFirst := errors.New("first error")

	c.Add("first", func() error {
		called = append(called, "first")
		return errFirst
	})
	c.Add("second", func() error {
		called = append(called, "second")
		return nil
	})

	err := c.Close()
	if err == nil {
		t.Fatal("Close() = nil, want non-nil error")
	}

	// Both functions must be called despite first (registered) erroring.
	if len(called) != 2 {
		t.Fatalf("expected 2 calls, got %d: %v", len(called), called)
	}

	// Error message must contain the name of the failing closer.
	if !strings.Contains(err.Error(), "first") {
		t.Errorf("error %q does not contain closer name %q", err.Error(), "first")
	}

	if !errors.Is(err, errFirst) {
		t.Errorf("errors.Is(err, errFirst) = false, want true")
	}
}
