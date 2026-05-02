package serve

import (
	"crypto/tls"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
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

const (
	defaultReadHeaderTimeout = 10 * time.Second
	defaultReadTimeout       = 30 * time.Second
	defaultWriteTimeout      = 60 * time.Second
	defaultIdleTimeout       = 120 * time.Second
	defaultMaxHeaderBytes    = 1 << 20 // 1 MiB
)

// InitialServerConfig returns production-grade server defaults.
func InitialServerConfig() ServerConfig {
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

// ServerConfigFromEnv returns a ServerConfig populated from environment variables,
// starting from InitialServerConfig() defaults. It reads SERVER_TRUSTED_PROXIES as a
// comma-separated list of CIDR prefixes. CIDR validity is deferred to ValidationRules.
func ServerConfigFromEnv() ServerConfig {
	cfg := InitialServerConfig()
	raw := strings.TrimSpace(os.Getenv("SERVER_TRUSTED_PROXIES"))
	if raw == "" {
		return cfg
	}
	for _, part := range strings.Split(raw, ",") {
		if cidr := strings.TrimSpace(part); cidr != "" {
			cfg.TrustedProxies = append(cfg.TrustedProxies, cidr)
		}
	}
	return cfg
}

// ValidationRules returns a validator registrar that checks each entry in
// TrustedProxies is a valid CIDR prefix.
func ValidationRules() func(*validator.Validate) {
	return func(v *validator.Validate) {
		v.RegisterStructValidation(serverConfigRules, ServerConfig{})
	}
}

func serverConfigRules(sl validator.StructLevel) {
	cfg := sl.Current().Interface().(ServerConfig)
	for _, cidr := range cfg.TrustedProxies {
		if _, err := ParseTrustedProxies([]string{cidr}); err != nil {
			sl.ReportError(cfg.TrustedProxies, "TrustedProxies", "TrustedProxies", "invalid_proxy_cidr", cidr)
			return
		}
	}
}
