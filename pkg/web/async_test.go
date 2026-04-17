package web

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// TestAsyncContext_StoresContextInUnderlyingCtx verifies that after the
// AsyncContext middleware runs, a deep-call function receiving only a
// context.Context can retrieve the same *Context instance via FromContext.
func TestAsyncContext_StoresContextInUnderlyingCtx(t *testing.T) {
	var captured *Context

	// Simulate a "deep" function that only receives context.Context.
	deepFn := func(ctx context.Context) {
		captured = FromContext(ctx)
	}

	handler := func(c *Context) (Response, error) {
		deepFn(c.ctx)
		return Respond(http.StatusOK), nil
	}

	mw := AsyncContext()
	wrapped := mw(handler)

	c := NewContext(context.Background(), Request{})
	wrapped(c) //nolint:errcheck

	if captured == nil {
		t.Fatal("FromContext returned nil; expected *Context")
	}
	if captured != c {
		t.Fatalf("FromContext returned a different *Context instance")
	}
}

// TestFromContext_CtxWithoutMiddleware verifies that FromContext returns nil
// when the AsyncContext middleware was never installed.
func TestFromContext_CtxWithoutMiddleware(t *testing.T) {
	got := FromContext(context.Background())
	if got != nil {
		t.Fatalf("FromContext(context.Background()) = %v, want nil", got)
	}
}

// TestFromContext_HandlerWithoutAsyncMiddleware verifies that FromContext
// returns nil even after a plain handler runs, if AsyncContext is not in the
// middleware chain.
func TestFromContext_HandlerWithoutAsyncMiddleware(t *testing.T) {
	var captured *Context

	handler := func(c *Context) (Response, error) {
		captured = FromContext(c.ctx)
		return Respond(http.StatusOK), nil
	}

	c := NewContext(context.Background(), Request{})
	handler(c) //nolint:errcheck

	if captured != nil {
		t.Fatalf("FromContext returned non-nil without AsyncContext middleware installed")
	}
}

// ---------------------------------------------------------------------------
// G9 — web.Go helper
// ---------------------------------------------------------------------------

// TestWebGo_RunsFn verifies that the function supplied to Go is executed.
func TestWebGo_RunsFn(t *testing.T) {
	done := make(chan struct{})

	Go(nil, "test", func() {
		close(done)
	})

	// Block until the goroutine signals completion.
	<-done
}

// syncLogger is a slog.Handler that wraps a bytes.Buffer with a mutex and
// signals a WaitGroup once the first log record is handled. This lets tests
// safely wait until the goroutine's panic log is fully written before reading.
type syncLogger struct {
	mu      sync.Mutex
	buf     bytes.Buffer
	once    sync.Once
	done    chan struct{}
	wrapped slog.Handler
}

func newSyncLogger(opts *slog.HandlerOptions) *syncLogger {
	sl := &syncLogger{done: make(chan struct{})}
	sl.wrapped = slog.NewTextHandler(&sl.buf, opts)
	return sl
}

func (s *syncLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return s.wrapped.Enabled(ctx, level)
}

func (s *syncLogger) Handle(ctx context.Context, r slog.Record) error {
	s.mu.Lock()
	err := s.wrapped.Handle(ctx, r)
	s.mu.Unlock()
	s.once.Do(func() { close(s.done) })
	return err
}

func (s *syncLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	return s.wrapped.WithAttrs(attrs)
}

func (s *syncLogger) WithGroup(name string) slog.Handler {
	return s.wrapped.WithGroup(name)
}

func (s *syncLogger) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// TestWebGo_RecoversPanic verifies that a panicking goroutine launched via Go
// does not crash the caller and that the caller is unaffected.
func TestWebGo_RecoversPanic(t *testing.T) {
	sl := newSyncLogger(&slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(sl)

	Go(logger, "test.panic", func() {
		panic("controlled panic in goroutine")
	})

	// Wait for the panic log to be written, confirming the goroutine ran and recovered.
	<-sl.done
	// If we reach here the panic was recovered and did not crash the caller.
}

// TestWebGo_LogsPanicGoroutineEvent verifies the log event shape emitted on panic.
func TestWebGo_LogsPanicGoroutineEvent(t *testing.T) {
	sl := newSyncLogger(&slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(sl)

	Go(logger, "my.subsystem", func() {
		panic("boom value")
	})

	// Wait until the panic log is fully written before reading.
	<-sl.done

	logged := sl.String()

	// message must be "panic.goroutine"
	if !strings.Contains(logged, "panic.goroutine") {
		t.Errorf("log must contain message 'panic.goroutine'; got:\n%s", logged)
	}
	// subsystem attr must match the passed label
	if !strings.Contains(logged, "subsystem=my.subsystem") {
		t.Errorf("log must contain 'subsystem=my.subsystem'; got:\n%s", logged)
	}
	// cause attr must be present and non-empty
	if !strings.Contains(logged, "cause=") {
		t.Errorf("log must contain 'cause=' attr; got:\n%s", logged)
	}
	// stack attr must be present and non-empty
	if !strings.Contains(logged, "stack=") {
		t.Errorf("log must contain 'stack=' attr; got:\n%s", logged)
	}
}

// TestWebGo_NilLoggerFallsBackToDefault verifies that a nil logger does not
// cause a crash; slog.Default() is used instead.
func TestWebGo_NilLoggerFallsBackToDefault(t *testing.T) {
	// Replace the default logger temporarily so the panic log goes somewhere
	// observable, then restore it.
	sl := newSyncLogger(&slog.HandlerOptions{Level: slog.LevelDebug})
	old := slog.Default()
	slog.SetDefault(slog.New(sl))
	t.Cleanup(func() { slog.SetDefault(old) })

	// nil logger — must fall back to slog.Default() and not crash.
	Go(nil, "nil.logger.test", func() {
		panic("nil logger panic")
	})

	// Wait until the panic is logged through the default logger.
	<-sl.done
	// Reaching here means no crash — test passes.
}
