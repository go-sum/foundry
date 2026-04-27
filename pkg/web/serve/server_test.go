package serve

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/web"
	"golang.org/x/net/http2"
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

func TestListenAndServe_ServeAndShutdown(t *testing.T) {
	// Pick a free port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close() //nolint:errcheck

	handler := func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "hello"), nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- ListenAndServe(ctx, handler, ServerConfig{
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
	resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		cancel()
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Cancel the context to trigger shutdown.
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("ListenAndServe returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("ListenAndServe did not return after context cancel")
	}
}

func TestListenAndServe_DefaultShutdownTimeout(t *testing.T) {
	// Pick a free port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close() //nolint:errcheck

	handler := func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		// ShutdownTimeout = 0 triggers the default 15s path.
		done <- ListenAndServe(ctx, handler, ServerConfig{Addr: addr})
	}()

	// Wait for the server to be ready.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, httpErr := http.Get(fmt.Sprintf("http://%s/", addr))
		if httpErr == nil {
			resp.Body.Close() //nolint:errcheck
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("ListenAndServe returned error: %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Error("ListenAndServe did not return after context cancel")
	}
}

// selfSignedTLSConfig generates a throwaway self-signed TLS certificate in memory
// and returns a *tls.Config configured with it.
func selfSignedTLSConfig(t *testing.T) *tls.Config {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Second),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}

	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	return &tls.Config{Certificates: []tls.Certificate{cert}}
}

func TestListenAndServe_TLS_ServeAndShutdown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close() //nolint:errcheck

	handler := func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "tls-ok"), nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- ListenAndServe(ctx, handler, ServerConfig{
			Addr:            addr,
			TLSConfig:       selfSignedTLSConfig(t),
			ShutdownTimeout: 5 * time.Second,
		})
	}()

	// Use a client that skips cert verification for the self-signed cert.
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test only
		},
	}

	var resp *http.Response
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err = client.Get(fmt.Sprintf("https://%s/", addr))
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		cancel()
		t.Fatalf("server did not start in time: %v", err)
	}
	resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		cancel()
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("ListenAndServe (TLS) returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("ListenAndServe (TLS) did not return after context cancel")
	}
}

func TestListenAndServe_TLS_HTTP2Negotiated(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close() //nolint:errcheck

	handler := func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "h2-ok"), nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- ListenAndServe(ctx, handler, ServerConfig{
			Addr:            addr,
			TLSConfig:       selfSignedTLSConfig(t),
			ShutdownTimeout: 5 * time.Second,
		})
	}()

	transport := &http2.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test only
	}
	client := &http.Client{Transport: transport}

	var resp *http.Response
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err = client.Get(fmt.Sprintf("https://%s/", addr))
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("server did not start in time: %v", err)
	}
	resp.Body.Close() //nolint:errcheck

	if resp.Proto != "HTTP/2.0" {
		t.Errorf("Proto = %q, want HTTP/2.0", resp.Proto)
	}
}

func TestNewServer_H2C_Handler(t *testing.T) {
	handler := func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	}

	srvWithout := NewServer(handler, ServerConfig{})
	srvWith := NewServer(handler, ServerConfig{H2C: true})

	// Both must have a non-nil handler; the h2c-wrapped one is a distinct type.
	if srvWithout.Handler == nil {
		t.Error("handler without H2C = nil")
	}
	if srvWith.Handler == nil {
		t.Error("handler with H2C = nil")
	}
	if srvWithout.Handler == srvWith.Handler {
		t.Error("H2C=true should produce a different handler wrapper")
	}
}

func TestListenAndServe_H2C_Cleartext(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close() //nolint:errcheck

	handler := func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "h2c-ok"), nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- ListenAndServe(ctx, handler, ServerConfig{
			Addr:            addr,
			H2C:             true,
			ShutdownTimeout: 5 * time.Second,
		})
	}()

	// h2c client connects over plain TCP.
	transport := &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
	}
	client := &http.Client{Transport: transport}

	var resp *http.Response
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err = client.Get(fmt.Sprintf("http://%s/", addr))
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		cancel()
		t.Fatalf("server did not start in time: %v", err)
	}
	resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		cancel()
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if resp.Proto != "HTTP/2.0" {
		t.Errorf("Proto = %q, want HTTP/2.0", resp.Proto)
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("ListenAndServe (H2C) returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("ListenAndServe (H2C) did not return after context cancel")
	}
}

func TestNewServer_TLSConfig(t *testing.T) {
	handler := func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	}
	input := &tls.Config{MinVersion: tls.VersionTLS13}
	srv := NewServer(handler, ServerConfig{TLSConfig: input})
	if srv.TLSConfig == nil {
		t.Fatal("TLSConfig not set on http.Server")
	}
	if srv.TLSConfig == input {
		t.Error("TLSConfig should be a clone, not the same pointer")
	}
}

func TestListenAndServe_H2C_TLS_MutuallyExclusive(t *testing.T) {
	handler := func(_ *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "ok"), nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ListenAndServe(ctx, handler, ServerConfig{
		H2C:       true,
		TLSConfig: &tls.Config{},
	})
	if err == nil {
		t.Fatal("expected error for H2C + TLSConfig, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error = %q, want substring 'mutually exclusive'", err)
	}
}
