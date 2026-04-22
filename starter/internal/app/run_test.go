package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	configpkg "github.com/go-sum/foundry/config"
	"github.com/go-sum/web"
	"github.com/go-sum/web/router"
	"github.com/go-sum/web/serve"
)

func TestRun_ShutsDownCleanlyOnContextCancel(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close() //nolint:errcheck

	rt := router.New()
	rt.GET("/health", "health", func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	})
	rt.Freeze()

	a := &App{
		Runtime: Runtime{
			Config: &configpkg.Config{
				Server: serve.ServerConfig{
					Addr:            addr,
					ShutdownTimeout: 5 * time.Second,
				},
			},
		},
		router: rt,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- a.Run(ctx)
	}()

	// Poll until the server is ready.
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
	httpResp.Body.Close() //nolint:errcheck

	if httpResp.StatusCode != http.StatusOK {
		cancel()
		t.Fatalf("status = %d, want %d", httpResp.StatusCode, http.StatusOK)
	}

	cancel()

	select {
	case runErr := <-done:
		if runErr != nil {
			t.Errorf("Run() error = %v, want nil", runErr)
		}
	case <-time.After(10 * time.Second):
		t.Error("Run did not return after context cancel")
	}
}

func TestRun_ContextCanceledImmediately(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close() //nolint:errcheck

	rt := router.New()
	rt.GET("/health", "health", func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	})
	rt.Freeze()

	a := &App{
		Runtime: Runtime{
			Config: &configpkg.Config{
				Server: serve.ServerConfig{
					Addr:            addr,
					ShutdownTimeout: 5 * time.Second,
				},
			},
		},
		router: rt,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan error, 1)
	go func() {
		done <- a.Run(ctx)
	}()

	select {
	case runErr := <-done:
		if runErr != nil {
			t.Errorf("Run() error = %v, want nil", runErr)
		}
	case <-time.After(5 * time.Second):
		t.Error("Run did not return after immediate context cancel")
	}
}
