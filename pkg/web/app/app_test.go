package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-sum/web"
	"github.com/go-sum/web/serve"
)

// stubErrorRenderer is a minimal ErrorRenderer for test assertions.
type stubErrorRenderer struct {
	called bool
}

func (s *stubErrorRenderer) RenderError(_ *web.Context, _ *web.Error) web.Response {
	s.called = true
	return web.Text(http.StatusInternalServerError, "error")
}

func TestNew_defaults(t *testing.T) {
	a := New(Config{})

	if a.Logger != slog.Default() {
		t.Error("Logger: want slog.Default(), got a different logger")
	}
	if a.Router == nil {
		t.Fatal("Router: want non-nil *router.Router, got nil")
	}
	if a.cfg.Server.Addr != ":8080" {
		t.Errorf("Server.Addr = %q, want %q", a.cfg.Server.Addr, ":8080")
	}
	// SecureDefaults zero value is false — the plan default is true only when
	// the caller sets it explicitly. The zero value of bool is false, so
	// NewWithoutSecureDefaults is used unless the caller sets SecureDefaults: true.
	// Verify the router is not nil and is functional.
	if a.Router.IsFrozen() {
		t.Error("Router should not be frozen before Run")
	}
}

func TestNew_secureDefaultsTrue(t *testing.T) {
	a := New(Config{SecureDefaults: true})

	if a.Router == nil {
		t.Fatal("Router: want non-nil *router.Router, got nil")
	}
	if a.Router.IsFrozen() {
		t.Error("Router should not be frozen before Run")
	}
}

func TestNew_secureDefaultsFalse(t *testing.T) {
	a := New(Config{SecureDefaults: false})

	if a.Router == nil {
		t.Fatal("Router: want non-nil *router.Router, got nil")
	}
}

func TestNew_withCustomLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil, nil))
	a := New(Config{Logger: logger})

	if a.Logger != logger {
		t.Error("Logger: want the provided logger, got a different value")
	}
	if a.cfg.Boundary.Logger != logger {
		t.Error("Boundary.Logger: want the provided logger, got a different value")
	}
}

func TestNew_withErrorRenderer(t *testing.T) {
	renderer := &stubErrorRenderer{}
	a := New(Config{ErrorRenderer: renderer})

	if a.cfg.Boundary.Renderer != renderer {
		t.Error("Boundary.Renderer: want the provided ErrorRenderer, got a different value")
	}
}

func TestNew_boundaryRendererNotOverriddenWhenAlreadySet(t *testing.T) {
	renderer1 := &stubErrorRenderer{}
	renderer2 := &stubErrorRenderer{}
	a := New(Config{
		ErrorRenderer: renderer1,
		Boundary: web.BoundaryConfig{
			Renderer: renderer2,
		},
	})

	// When Boundary.Renderer is already set, it must not be overridden.
	if a.cfg.Boundary.Renderer != renderer2 {
		t.Error("Boundary.Renderer: want the explicitly set renderer2, got overridden by ErrorRenderer")
	}
}

func TestNew_boundaryLoggerNotOverriddenWhenAlreadySet(t *testing.T) {
	logger1 := slog.New(slog.NewTextHandler(nil, nil))
	logger2 := slog.New(slog.NewTextHandler(nil, nil))
	a := New(Config{
		Logger: logger1,
		Boundary: web.BoundaryConfig{
			Logger: logger2,
		},
	})

	// When Boundary.Logger is already set, it must not be overridden.
	if a.cfg.Boundary.Logger != logger2 {
		t.Error("Boundary.Logger: want the explicitly set logger2, got overridden by Config.Logger")
	}
}

func TestNew_serverAddrDefaultsWhenEmpty(t *testing.T) {
	a := New(Config{})
	if a.cfg.Server.Addr != ":8080" {
		t.Errorf("Server.Addr = %q, want %q (DefaultServerConfig)", a.cfg.Server.Addr, ":8080")
	}
}

func TestNew_serverAddrPreservedWhenSet(t *testing.T) {
	a := New(Config{Server: serve.ServerConfig{Addr: ":9090"}})
	if a.cfg.Server.Addr != ":9090" {
		t.Errorf("Server.Addr = %q, want %q", a.cfg.Server.Addr, ":9090")
	}
}

func TestApp_Run_serveAndShutdown(t *testing.T) {
	// Pick a free port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	a := New(Config{
		Server: serve.ServerConfig{
			Addr:            addr,
			ShutdownTimeout: 5 * time.Second,
		},
	})

	// Register a simple health-check route.
	a.Router.GET("/health", "health.show", func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- a.Run(ctx)
	}()

	// Wait for the server to become ready.
	var httpResp *http.Response
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		httpResp, err = http.Get(fmt.Sprintf("http://%s/health", addr))
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		cancel()
		t.Fatalf("server did not start in time: %v", err)
	}
	httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		cancel()
		t.Fatalf("status = %d, want %d", httpResp.StatusCode, http.StatusOK)
	}

	// Cancel the context to trigger graceful shutdown.
	cancel()

	select {
	case runErr := <-done:
		if runErr != nil {
			t.Errorf("Run returned error: %v", runErr)
		}
	case <-time.After(10 * time.Second):
		t.Error("Run did not return after context cancel")
	}
}

func TestApp_Run_contextCanceledImmediately(t *testing.T) {
	// Pick a free port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	a := New(Config{
		Server: serve.ServerConfig{
			Addr:            addr,
			ShutdownTimeout: 2 * time.Second,
		},
	})

	// Cancel context before Run has a chance to accept connections.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan error, 1)
	go func() {
		done <- a.Run(ctx)
	}()

	select {
	case runErr := <-done:
		if runErr != nil {
			t.Errorf("Run returned error on immediate cancel: %v", runErr)
		}
	case <-time.After(10 * time.Second):
		t.Error("Run did not return after immediate context cancel")
	}
}
