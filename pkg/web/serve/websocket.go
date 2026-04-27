package serve

import (
	"bufio"
	"net"
	"net/http"

	"github.com/go-sum/foundry/pkg/web"
)

// HijackFunc is called with the raw TCP connection and buffered reader/writer
// after a successful 101 Switching Protocols WebSocket upgrade. The caller is
// responsible for the connection lifecycle from this point on.
type HijackFunc func(conn net.Conn, brw *bufio.ReadWriter) error

// hijackBody is stored as the Response.Body for a switching-protocols response.
// It carries the HijackFunc through WriteHTTPResponse.
type hijackBody struct {
	fn HijackFunc
}

func (h *hijackBody) Read(p []byte) (int, error) { return 0, nil }
func (h *hijackBody) Close() error               { return nil }

// Switching returns a web.Response with status 101 Switching Protocols that,
// when written by the adapt package, hijacks the connection and calls fn.
//
// The handler should set the required upgrade headers before returning:
//
//	resp := adapt.Switching(fn)
//	resp.Headers.Set("Upgrade", "websocket")
//	resp.Headers.Set("Connection", "Upgrade")
//	resp.Headers.Set("Sec-WebSocket-Accept", ...)
//	return resp
func Switching(fn HijackFunc) web.Response {
	resp := web.Respond(http.StatusSwitchingProtocols)
	resp.Body = &hijackBody{fn: fn}
	return resp
}
