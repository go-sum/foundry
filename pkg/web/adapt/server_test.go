package adapt

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-sum/web"
)

func TestNewServer_DefaultTimeouts(t *testing.T) {
	handler := func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	}

	srv := NewServer("", handler, ServerOptions{})

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

	opts := ServerOptions{
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    512 * 1024,
	}

	srv := NewServer(":9090", handler, opts)

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

	srv := NewServer("", handler, ServerOptions{})
	if srv.Handler == nil {
		t.Error("Handler = nil, want non-nil http.Handler")
	}
}

func TestShutdown_NotListening(t *testing.T) {
	handler := func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	}

	srv := NewServer("", handler, ServerOptions{})
	err := Shutdown(context.Background(), srv)
	if err != nil {
		t.Errorf("Shutdown returned error for non-listening server: %v", err)
	}
}
