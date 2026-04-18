package serve

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-sum/web"
)

const (
	defaultReadHeaderTimeout = 10 * time.Second
	defaultReadTimeout       = 30 * time.Second
	defaultWriteTimeout      = 60 * time.Second
	defaultIdleTimeout       = 120 * time.Second
	defaultMaxHeaderBytes    = 1 << 20 // 1 MiB
)

// ServerConfig configures NewServer.
type ServerConfig struct {
	// Addr is the TCP address for the server to listen on. Defaults to ":8080".
	Addr string
	// ReadHeaderTimeout is the max time to read request headers. Defaults to 10s.
	ReadHeaderTimeout time.Duration
	// ReadTimeout is the max time to read the full request (headers + body). Defaults to 30s.
	ReadTimeout time.Duration
	// WriteTimeout is the max time to write a response. Defaults to 60s.
	WriteTimeout time.Duration
	// IdleTimeout is the max time an idle keep-alive connection may linger. Defaults to 120s.
	IdleTimeout time.Duration
	// ShutdownTimeout is the max time to wait for active connections to drain on shutdown.
	ShutdownTimeout time.Duration
	// MaxHeaderBytes limits the request header size. Defaults to 1 MiB.
	MaxHeaderBytes int
	// TrustedProxies lists CIDR prefixes of trusted reverse proxies.
	TrustedProxies []string
	// ErrorLog is used for http.Server.ErrorLog. If nil, output goes to stderr via log package.
	ErrorLog interface{ Printf(format string, v ...any) }
}

// DefaultServerConfig returns production-grade server defaults.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Addr:              ":8080",
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ShutdownTimeout:   15 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

// NewServer creates a *http.Server with handler adapted via ToHTTPHandler, using
// production-safe timeouts and the given config. cfg.Addr defaults to ":8080".
//
// Use Shutdown to drain active connections gracefully.
func NewServer(handler web.Handler, cfg ServerConfig) *http.Server {
	addr := cfg.Addr
	if addr == "" {
		addr = ":8080"
	}

	readHeaderTimeout := cfg.ReadHeaderTimeout
	if readHeaderTimeout == 0 {
		readHeaderTimeout = defaultReadHeaderTimeout
	}

	readTimeout := cfg.ReadTimeout
	if readTimeout == 0 {
		readTimeout = defaultReadTimeout
	}

	writeTimeout := cfg.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = defaultWriteTimeout
	}

	idleTimeout := cfg.IdleTimeout
	if idleTimeout == 0 {
		idleTimeout = defaultIdleTimeout
	}

	maxHeaderBytes := cfg.MaxHeaderBytes
	if maxHeaderBytes == 0 {
		maxHeaderBytes = defaultMaxHeaderBytes
	}

	var errorLog *log.Logger
	if cfg.ErrorLog != nil {
		errorLog = log.New(logWriter{cfg.ErrorLog}, "", 0)
	}

	return &http.Server{
		Addr:              addr,
		Handler:           ToHTTPHandler(handler),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          errorLog,
	}
}

// Shutdown gracefully drains the server within the context deadline, then closes it.
func Shutdown(ctx context.Context, srv *http.Server) error {
	return srv.Shutdown(ctx)
}

// logWriter bridges the ErrorLog interface to log.Logger's io.Writer interface.
type logWriter struct {
	l interface{ Printf(format string, v ...any) }
}

func (w logWriter) Write(p []byte) (int, error) {
	w.l.Printf("%s", p)
	return len(p), nil
}
