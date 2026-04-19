package web

import (
	"context"
	"net/http"
	"testing"
)

func TestChain(t *testing.T) {
	var order []string

	mwA := func(next Handler) Handler {
		return func(c *Context) (Response, error) {
			order = append(order, "A-before")
			resp, err := next(c)
			order = append(order, "A-after")
			return resp, err
		}
	}
	mwB := func(next Handler) Handler {
		return func(c *Context) (Response, error) {
			order = append(order, "B-before")
			resp, err := next(c)
			order = append(order, "B-after")
			return resp, err
		}
	}
	base := func(_ *Context) (Response, error) {
		order = append(order, "handler")
		return Respond(http.StatusOK), nil
	}

	h := Chain(base, mwA, mwB)
	resp, err := h(NewContext(context.Background(), Request{}))

	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Status, http.StatusOK)
	}

	want := []string{"A-before", "B-before", "handler", "B-after", "A-after"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q", i, order[i], want[i])
		}
	}
}

func TestNotFoundHandler(t *testing.T) {
	_, err := NotFoundHandler()(NewContext(context.Background(), Request{}))
	if err == nil {
		t.Fatal("err = nil, want non-nil")
	}
	e, ok := err.(*Error)
	if !ok {
		t.Fatalf("err type = %T, want *Error", err)
	}
	if e.Status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", e.Status, http.StatusNotFound)
	}
}

func TestMethodNotAllowedHandler(t *testing.T) {
	_, err := MethodNotAllowedHandler()(NewContext(context.Background(), Request{}))
	if err == nil {
		t.Fatal("err = nil, want non-nil")
	}
	e, ok := err.(*Error)
	if !ok {
		t.Fatalf("err type = %T, want *Error", err)
	}
	if e.Status != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", e.Status, http.StatusMethodNotAllowed)
	}
}

func TestCheckCancellation_ActiveContext_CallsNext(t *testing.T) {
	called := false
	inner := func(_ *Context) (Response, error) {
		called = true
		return Respond(http.StatusOK), nil
	}
	h := Chain(inner, CheckCancellation())
	_, err := h(NewContext(context.Background(), Request{}))
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if !called {
		t.Fatal("inner handler was not called")
	}
}

func TestCheckCancellation_CancelledContext_ShortCircuits(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	inner := func(_ *Context) (Response, error) {
		called = true
		return Respond(http.StatusOK), nil
	}
	h := Chain(inner, CheckCancellation())
	_, err := h(NewContext(ctx, Request{}))
	if err != context.Canceled {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if called {
		t.Fatal("inner handler should not have been called")
	}
}

func TestCheckCancellation_DeadlineExceeded_ShortCircuits(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	called := false
	inner := func(_ *Context) (Response, error) {
		called = true
		return Respond(http.StatusOK), nil
	}
	h := Chain(inner, CheckCancellation())
	_, err := h(NewContext(ctx, Request{}))
	if err != context.DeadlineExceeded {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
	if called {
		t.Fatal("inner handler should not have been called")
	}
}

func TestWithRequestID_SetsHeader(t *testing.T) {
	inner := func(_ *Context) (Response, error) {
		return Respond(http.StatusOK), nil
	}
	h := Chain(inner, WithRequestID())
	resp, err := h(NewContext(context.Background(), Request{}))
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	id := resp.Headers.Get("X-Request-Id")
	if id == "" {
		t.Fatal("X-Request-Id header is empty")
	}
	if len(id) != 32 {
		t.Errorf("X-Request-Id length = %d, want 32", len(id))
	}
}

func TestWithRequestID_StoresInContext(t *testing.T) {
	var captured string
	inner := func(c *Context) (Response, error) {
		captured = RequestID(c)
		return Respond(http.StatusOK), nil
	}
	h := Chain(inner, WithRequestID())
	resp, err := h(NewContext(context.Background(), Request{}))
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if captured == "" {
		t.Fatal("RequestID in context is empty")
	}
	if captured != resp.Headers.Get("X-Request-Id") {
		t.Errorf("context ID %q != header ID %q", captured, resp.Headers.Get("X-Request-Id"))
	}
}
