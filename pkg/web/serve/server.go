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
	"time"

	"github.com/go-sum/foundry/pkg/web"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// logWriter bridges the ErrorLog interface to log.Logger's io.Writer interface.
type logWriter struct {
	l interface{ Printf(format string, v ...any) }
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

func (w logWriter) Write(p []byte) (int, error) {
	w.l.Printf("%s", p)
	return len(p), nil
}
