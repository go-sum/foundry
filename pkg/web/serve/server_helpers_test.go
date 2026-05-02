package serve

import (
	"fmt"
	"testing"
	"time"
)

type recordingErrorLog struct {
	lines []string
}

func (l *recordingErrorLog) Printf(format string, v ...any) {
	l.lines = append(l.lines, fmt.Sprintf(format, v...))
}

func TestInitialServerConfig(t *testing.T) {
	got := InitialServerConfig()
	if got.Addr != ":8080" {
		t.Fatalf("Addr = %q, want %q", got.Addr, ":8080")
	}
	if got.ReadHeaderTimeout != 10*time.Second {
		t.Fatalf("ReadHeaderTimeout = %v, want %v", got.ReadHeaderTimeout, 10*time.Second)
	}
	if got.ReadTimeout != 30*time.Second {
		t.Fatalf("ReadTimeout = %v, want %v", got.ReadTimeout, 30*time.Second)
	}
	if got.WriteTimeout != 60*time.Second {
		t.Fatalf("WriteTimeout = %v, want %v", got.WriteTimeout, 60*time.Second)
	}
	if got.IdleTimeout != 120*time.Second {
		t.Fatalf("IdleTimeout = %v, want %v", got.IdleTimeout, 120*time.Second)
	}
	if got.ShutdownTimeout != 15*time.Second {
		t.Fatalf("ShutdownTimeout = %v, want %v", got.ShutdownTimeout, 15*time.Second)
	}
	if got.MaxHeaderBytes != 1<<20 {
		t.Fatalf("MaxHeaderBytes = %d, want %d", got.MaxHeaderBytes, 1<<20)
	}
}

func TestLogWriter_Write(t *testing.T) {
	logger := &recordingErrorLog{}
	n, err := (logWriter{l: logger}).Write([]byte("server error"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len("server error") {
		t.Fatalf("Write() bytes = %d, want %d", n, len("server error"))
	}
	if len(logger.lines) != 1 || logger.lines[0] != "server error" {
		t.Fatalf("logged lines = %#v, want [\"server error\"]", logger.lines)
	}
}
