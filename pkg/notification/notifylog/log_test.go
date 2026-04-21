package notifylog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/go-sum/notification"
	"github.com/go-sum/notification/notifylog"
)

func TestSender_Send_EmitsStructuredLog(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	s := notifylog.New(logger)
	n := notification.Notification{
		ID:      "test-id-123",
		Subject: "alert subject",
		Body:    "alert body",
		Correlation: notification.Correlation{
			RequestID: "req-456",
			TraceID:   "trace-789",
			Op:        "test.op",
		},
	}

	if err := s.Send(context.Background(), n); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("log buffer is empty after Send")
	}

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("log output is not valid JSON: %v\noutput: %s", err, buf.String())
	}

	// msg field
	if got, ok := record["msg"]; !ok || got != "notification.send" {
		t.Errorf("msg = %v, want %q", got, "notification.send")
	}
	// id field
	if got, ok := record["id"]; !ok || got != "test-id-123" {
		t.Errorf("id = %v, want %q", got, "test-id-123")
	}
	// subject field
	if got, ok := record["subject"]; !ok || got != "alert subject" {
		t.Errorf("subject = %v, want %q", got, "alert subject")
	}
	// body field
	if got, ok := record["body"]; !ok || got != "alert body" {
		t.Errorf("body = %v, want %q", got, "alert body")
	}
	// request_id field
	if got, ok := record["request_id"]; !ok || got != "req-456" {
		t.Errorf("request_id = %v, want %q", got, "req-456")
	}
	// trace_id field
	if got, ok := record["trace_id"]; !ok || got != "trace-789" {
		t.Errorf("trace_id = %v, want %q", got, "trace-789")
	}
	// op field
	if got, ok := record["op"]; !ok || got != "test.op" {
		t.Errorf("op = %v, want %q", got, "test.op")
	}
}

func TestNew_NilLogger_UsesSlogDefault(t *testing.T) {
	// Should not panic.
	s := notifylog.New(nil)
	n := notification.Notification{Subject: "safe"}
	if err := s.Send(context.Background(), n); err != nil {
		t.Errorf("Send returned error: %v", err)
	}
}
