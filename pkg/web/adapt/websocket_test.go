package adapt_test

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-sum/web"
	"github.com/go-sum/web/adapt"
)

// TestSwitching_HijackCalled verifies that a 101 response causes the
// connection to be hijacked and the HijackFunc to be invoked with a non-nil
// conn and brw. Uses a real net.Conn via httptest.NewServer.
func TestSwitching_HijackCalled(t *testing.T) {
	hijackCalled := make(chan struct{}, 1)

	fn := func(conn net.Conn, brw *bufio.ReadWriter) error {
		if conn == nil {
			t.Error("HijackFunc: conn is nil")
		}
		if brw == nil {
			t.Error("HijackFunc: brw is nil")
		}
		conn.Close()
		close(hijackCalled)
		return nil
	}

	handler := func(c *web.Context) (web.Response, error) {
		resp := adapt.Switching(fn)
		resp.Headers.Set("Upgrade", "websocket")
		resp.Headers.Set("Connection", "Upgrade")
		return resp, nil
	}

	srv := httptest.NewServer(adapt.ToHTTPHandler(handler))
	defer srv.Close()

	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Send a minimal HTTP/1.1 GET request.
	_, err = conn.Write([]byte("GET / HTTP/1.1\r\nHost: localhost\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"))
	if err != nil {
		t.Fatalf("write request: %v", err)
	}

	// Read the response status line to confirm 101.
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusSwitchingProtocols)
	}
	if got := resp.Header.Get("Upgrade"); got != "websocket" {
		t.Errorf("Upgrade header = %q, want %q", got, "websocket")
	}

	<-hijackCalled
}

// TestSwitching_NilFn verifies that a nil HijackFunc does not panic and
// reports an error via OnError instead of hanging.
func TestSwitching_NilFn(t *testing.T) {
	var errs []error
	resp := adapt.Switching(nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	adapt.WriteHTTPResponse(rec, req, resp, adapt.Config{
		OnError: func(err error) {
			errs = append(errs, err)
		},
	})

	if len(errs) != 1 {
		t.Fatalf("OnError call count = %d, want 1", len(errs))
	}
	if got := errs[0].Error(); got == "" {
		t.Error("OnError received empty error")
	}
}

// TestNon101ResponseRegression verifies that normal (non-WebSocket) responses
// still work correctly after the hijack block was introduced.
func TestNon101ResponseRegression(t *testing.T) {
	handler := func(c *web.Context) (web.Response, error) {
		return web.Text(http.StatusOK, "hello"), nil
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	adapt.ToHTTPHandler(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "hello" {
		t.Fatalf("body = %q, want %q", got, "hello")
	}
}

// TestSwitching_ErrorPropagated verifies that an error returned by HijackFunc
// is passed to the OnError callback.
func TestSwitching_ErrorPropagated(t *testing.T) {
	sentinel := errors.New("ws handler error")
	errCh := make(chan error, 1)

	fn := func(conn net.Conn, brw *bufio.ReadWriter) error {
		conn.Close()
		return sentinel
	}

	handler := func(c *web.Context) (web.Response, error) {
		resp := adapt.Switching(fn)
		resp.Headers.Set("Upgrade", "websocket")
		resp.Headers.Set("Connection", "Upgrade")
		return resp, nil
	}

	srv := httptest.NewServer(adapt.ToHTTPHandlerWithConfig(handler, adapt.Config{
		OnError: func(err error) {
			select {
			case errCh <- err:
			default:
			}
		},
	}))
	defer srv.Close()

	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_, _ = conn.Write([]byte("GET / HTTP/1.1\r\nHost: localhost\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"))

	br := bufio.NewReader(conn)
	_, _ = http.ReadResponse(br, nil)
	// Close client side so the hijacked conn.Close() in fn propagates.
	conn.Close()

	gotErr := <-errCh
	if gotErr == nil {
		t.Fatal("OnError was not called")
	}
	if !errors.Is(gotErr, sentinel) {
		t.Errorf("OnError err = %v, want wrapping %v", gotErr, sentinel)
	}
}
