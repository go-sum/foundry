package worker

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeProcessor struct {
	started   chan struct{}
	stopCount int
	pingErr   error
}

func newFakeProcessor() *fakeProcessor {
	return &fakeProcessor{started: make(chan struct{})}
}

func (p *fakeProcessor) Start(context.Context) {
	select {
	case p.started <- struct{}{}:
	default:
	}
}

func (p *fakeProcessor) Ping(context.Context) error {
	return p.pingErr
}

func (p *fakeProcessor) Stop() error {
	p.stopCount++
	return nil
}

func setupTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("EMAIL_PROVIDER", "log")
}

func TestWorkerRun_StartsProcessorAndStopsOnCancel(t *testing.T) {
	setupTestEnv(t)
	processor := newFakeProcessor()

	w, err := New(context.Background(), WithServicesFactory(func(context.Context, Runtime) (Services, error) {
		return Services{Processor: processor}, nil
	}))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Errorf("Worker.Close() error = %v", err)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- w.Run(ctx)
	}()

	select {
	case <-processor.started:
		// processor was started
	case <-time.After(2 * time.Second):
		t.Fatal("processor was not started within timeout")
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not return after cancel")
	}
}

func TestWorkerCheck_UsesProcessorPing(t *testing.T) {
	setupTestEnv(t)
	wantErr := errors.New("queue unavailable")
	processor := newFakeProcessor()
	processor.pingErr = wantErr

	w, err := New(context.Background(), WithServicesFactory(func(context.Context, Runtime) (Services, error) {
		return Services{Processor: processor}, nil
	}))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Errorf("Worker.Close() error = %v", err)
		}
	})

	err = w.Check(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("Check() error = %v, want %v", err, wantErr)
	}
}

func TestWorker_Run_NilProcessor_ReturnsError(t *testing.T) {
	setupTestEnv(t)

	w, err := New(context.Background(), WithServicesFactory(func(context.Context, Runtime) (Services, error) {
		return Services{Processor: nil}, nil
	}))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Errorf("Worker.Close() error = %v", err)
		}
	})

	err = w.Run(context.Background())
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil error for nil processor")
	}
}

func TestWorker_Close_StopsProcessor(t *testing.T) {
	setupTestEnv(t)
	processor := newFakeProcessor()

	w, err := New(context.Background(), WithServicesFactory(func(context.Context, Runtime) (Services, error) {
		return Services{Processor: processor}, nil
	}))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Worker.Close() error = %v", err)
	}
	if processor.stopCount != 1 {
		t.Fatalf("fakeProcessor.stopCount = %d, want 1", processor.stopCount)
	}
}
