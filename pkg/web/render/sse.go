package render

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-sum/foundry/pkg/web"
)

// SSEEvent is a single server-sent event.
type SSEEvent struct {
	// ID is the event's last-event-id (optional).
	ID string
	// Event is the event type name (optional; default is "message").
	Event string
	// Data is the event payload. Newlines are handled automatically.
	Data string
	// Retry is the reconnection time in milliseconds (optional; 0 = omit).
	Retry int
}

// Encode writes the SSE wire format for this event to w.
// The event block is terminated by a blank line as per the SSE spec.
func (e SSEEvent) Encode(w io.Writer) error {
	if e.ID != "" {
		if _, err := fmt.Fprintf(w, "id: %s\n", e.ID); err != nil {
			return err
		}
	}
	if e.Event != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", e.Event); err != nil {
			return err
		}
	}
	if e.Retry > 0 {
		if _, err := fmt.Fprintf(w, "retry: %d\n", e.Retry); err != nil {
			return err
		}
	}
	// Each line of Data must be prefixed with "data: ".
	lines := strings.Split(e.Data, "\n")
	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

// SSEWriter sends server-sent events to a connected client.
// Obtain one via NewSSEResponse; write events until the context is cancelled.
type SSEWriter struct {
	pw *io.PipeWriter
}

// Send writes an event to the SSE stream. Returns an error if the client disconnected.
func (w *SSEWriter) Send(event SSEEvent) error {
	return event.Encode(w.pw)
}

// Close closes the SSE stream (signals EOF to the client).
func (w *SSEWriter) Close() error {
	return w.pw.Close()
}

// NewSSEResponse creates an SSE streaming response and a writer for sending events.
// The caller must call writer.Close() when done to release the response body.
//
// Typical usage:
//
//	resp, sse := render.NewSSEResponse()
//	go func() {
//	    defer sse.Close()
//	    for event := range events {
//	        if err := sse.Send(event); err != nil { return }
//	    }
//	}()
//	return resp
func NewSSEResponse() (web.Response, *SSEWriter) {
	pr, pw := io.Pipe()
	h := web.NewHeaders()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no") // disable nginx buffering
	return web.Response{
		Status:  http.StatusOK,
		Headers: h,
		Body:    pr,
	}, &SSEWriter{pw: pw}
}
