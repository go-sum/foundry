package serve

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-sum/web"
)

func TestNewServer_DefaultTimeouts(t *testing.T) {
	handler := func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	}

	srv := NewServer(handler, ServerConfig{})

	if srv.Addr != ":8080" {
		t.Errorf("Addr = %q, want %q", srv.Addr, ":8080")
	}
	if srv.ReadHeaderTimeout != 10*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want %v", srv.ReadHeaderTimeout, 10*time.Second)
	}
	if srv.ReadTimeout != 30*time.Second {
		t.Errorf("ReadTimeout = %v, want %v", srv.ReadTimeout, 30*time.Second)
	}
	if srv.WriteTimeout != 60*time.Second {
		t.Errorf("WriteTimeout = %v, want %v", srv.WriteTimeout, 60*time.Second)
	}
	if srv.IdleTimeout != 120*time.Second {
		t.Errorf("IdleTimeout = %v, want %v", srv.IdleTimeout, 120*time.Second)
	}
	if srv.MaxHeaderBytes != 1<<20 {
		t.Errorf("MaxHeaderBytes = %d, want %d", srv.MaxHeaderBytes, 1<<20)
	}
}

func TestNewServer_CustomTimeouts(t *testing.T) {
	handler := func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	}

	cfg := ServerConfig{
		Addr:              ":9090",
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    512 * 1024,
	}

	srv := NewServer(handler, cfg)

	if srv.Addr != ":9090" {
		t.Errorf("Addr = %q, want %q", srv.Addr, ":9090")
	}
	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want %v", srv.ReadHeaderTimeout, 5*time.Second)
	}
	if srv.ReadTimeout != 15*time.Second {
		t.Errorf("ReadTimeout = %v, want %v", srv.ReadTimeout, 15*time.Second)
	}
	if srv.WriteTimeout != 30*time.Second {
		t.Errorf("WriteTimeout = %v, want %v", srv.WriteTimeout, 30*time.Second)
	}
	if srv.IdleTimeout != 60*time.Second {
		t.Errorf("IdleTimeout = %v, want %v", srv.IdleTimeout, 60*time.Second)
	}
	if srv.MaxHeaderBytes != 512*1024 {
		t.Errorf("MaxHeaderBytes = %d, want %d", srv.MaxHeaderBytes, 512*1024)
	}
}

func TestNewServer_HandlerNotNil(t *testing.T) {
	handler := func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	}

	srv := NewServer(handler, ServerConfig{})
	if srv.Handler == nil {
		t.Error("Handler = nil, want non-nil http.Handler")
	}
}

func TestShutdown_NotListening(t *testing.T) {
	handler := func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	}

	srv := NewServer(handler, ServerConfig{})
	err := Shutdown(context.Background(), srv)
	if err != nil {
		t.Errorf("Shutdown returned error for non-listening server: %v", err)
	}
}

func TestListenAndServeGracefully_ServeAndShutdown(t *testing.T) {
	// Pick a free port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	handler := func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "hello"), nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- ListenAndServeGracefully(ctx, handler, ServerConfig{
			Addr:            addr,
			ShutdownTimeout: 5 * time.Second,
		})
	}()

	// Wait for the server to accept connections.
	var resp *http.Response
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err = http.Get(fmt.Sprintf("http://%s/", addr))
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		cancel()
		t.Fatalf("server did not start in time: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cancel()
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Cancel the context to trigger shutdown.
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("ListenAndServeGracefully returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("ListenAndServeGracefully did not return after context cancel")
	}
}

func TestListenAndServeGracefully_DefaultShutdownTimeout(t *testing.T) {
	// Pick a free port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	handler := func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		// ShutdownTimeout = 0 triggers the default 15s path.
		done <- ListenAndServeGracefully(ctx, handler, ServerConfig{Addr: addr})
	}()

	// Wait for the server to be ready.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, httpErr := http.Get(fmt.Sprintf("http://%s/", addr))
		if httpErr == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("ListenAndServeGracefully returned error: %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Error("ListenAndServeGracefully did not return after context cancel")
	}
}
