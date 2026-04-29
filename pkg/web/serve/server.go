package serve

import (
	"cmp"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-sum/foundry/pkg/web"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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
	// H2C enables cleartext HTTP/2 (h2c) by wrapping the handler with h2c.NewHandler.
	// Use this when terminating TLS at a load balancer that forwards plain HTTP/2.
	H2C bool
	// TLSConfig, when non-nil, enables HTTPS. HTTP/2 is negotiated automatically via ALPN.
	// Mutually exclusive with H2C.
	TLSConfig *tls.Config
	// ErrorLog is used for http.Server.ErrorLog. If nil, output goes to stderr via log package.
	ErrorLog interface{ Printf(format string, v ...any) }
}

// DefaultServerConfig returns production-grade server defaults.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Addr:              ":8080",
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		ShutdownTimeout:   15 * time.Second,
		MaxHeaderBytes:    defaultMaxHeaderBytes,
	}
}

// DefaultServerConfigFromEnv returns a ServerConfig populated from environment variables,
// starting from DefaultServerConfig() defaults. It reads SERVER_TRUSTED_PROXIES as a
// comma-separated list of CIDR prefixes and validates each entry.
func DefaultServerConfigFromEnv() (ServerConfig, error) {
	cfg := DefaultServerConfig()
	raw := strings.TrimSpace(os.Getenv("SERVER_TRUSTED_PROXIES"))
	if raw == "" {
		return cfg, nil
	}
	parts := strings.Split(raw, ",")
	var proxies []string
	for _, part := range parts {
		if cidr := strings.TrimSpace(part); cidr != "" {
			proxies = append(proxies, cidr)
		}
	}
	if len(proxies) > 0 {
		if _, err := ParseTrustedProxies(proxies); err != nil {
			return ServerConfig{}, fmt.Errorf("serve: trusted proxies: %w", err)
		}
		cfg.TrustedProxies = proxies
	}
	return cfg, nil
}

// NewServer creates a *http.Server with handler adapted via ToHTTPHandler, using
// production-safe timeouts and the given config. cfg.Addr defaults to ":8080".
//
// Use Shutdown to drain active connections gracefully.
func NewServer(handler web.Handler, cfg ServerConfig) (*http.Server, error) {
	addr := cmp.Or(cfg.Addr, ":8080")
	readHeaderTimeout := cmp.Or(cfg.ReadHeaderTimeout, defaultReadHeaderTimeout)
	readTimeout := cmp.Or(cfg.ReadTimeout, defaultReadTimeout)
	writeTimeout := cmp.Or(cfg.WriteTimeout, defaultWriteTimeout)
	idleTimeout := cmp.Or(cfg.IdleTimeout, defaultIdleTimeout)
	maxHeaderBytes := cmp.Or(cfg.MaxHeaderBytes, defaultMaxHeaderBytes)

	var errorLog *log.Logger
	if cfg.ErrorLog != nil {
		errorLog = log.New(logWriter{cfg.ErrorLog}, "", 0)
	}

	var tlsCfg *tls.Config
	if cfg.TLSConfig != nil {
		tlsCfg = cfg.TLSConfig.Clone()
	}

	trustedProxies, err := ParseTrustedProxies(cfg.TrustedProxies)
	if err != nil {
		return nil, fmt.Errorf("web/serve: NewServer: %w", err)
	}
	httpHandler := http.Handler(ToHTTPHandlerWithConfig(handler, Config{TrustedProxies: trustedProxies}))
	if cfg.H2C {
		httpHandler = h2c.NewHandler(httpHandler, &http2.Server{})
	}

	return &http.Server{
		Addr:              addr,
		Handler:           httpHandler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          errorLog,
		TLSConfig:         tlsCfg,
	}, nil
}

// Shutdown gracefully drains the server within the context deadline, then closes it.
func Shutdown(ctx context.Context, srv *http.Server) error {
	return srv.Shutdown(ctx)
}

// ListenAndServe starts the HTTP or HTTPS server and blocks until ctx is
// canceled, then gracefully shuts down within cfg.ShutdownTimeout (defaulting to 15
// seconds). When cfg.TLSConfig is non-nil, the server listens over TLS and HTTP/2 is
// negotiated automatically via ALPN. H2C and TLSConfig are mutually exclusive.
// Signal handling is the caller's responsibility — use signal.NotifyContext in main.
func ListenAndServe(ctx context.Context, handler web.Handler, cfg ServerConfig) error {
	if cfg.H2C && cfg.TLSConfig != nil {
		return fmt.Errorf("web/serve: H2C and TLSConfig are mutually exclusive")
	}

	srv, err := NewServer(handler, cfg)
	if err != nil {
		return err
	}

	if srv.TLSConfig != nil {
		if err := http2.ConfigureServer(srv, nil); err != nil {
			return fmt.Errorf("web/serve: configure http2: %w", err)
		}
	}

	serveErr := make(chan error, 1)
	go func() {
		var err error
		if srv.TLSConfig != nil {
			var ln net.Listener
			ln, err = net.Listen("tcp", srv.Addr)
			if err == nil {
				err = srv.Serve(tls.NewListener(ln, srv.TLSConfig))
			}
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	timeout := cmp.Or(cfg.ShutdownTimeout, 15*time.Second)
	shutCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := Shutdown(shutCtx, srv); err != nil {
		return fmt.Errorf("web/serve: shutdown: %w", err)
	}
	if err, ok := <-serveErr; ok {
		return fmt.Errorf("web/serve: listen: %w", err)
	}
	return nil
}

// logWriter bridges the ErrorLog interface to log.Logger's io.Writer interface.
type logWriter struct {
	l interface{ Printf(format string, v ...any) }
}

func (w logWriter) Write(p []byte) (int, error) {
	w.l.Printf("%s", p)
	return len(p), nil
}
