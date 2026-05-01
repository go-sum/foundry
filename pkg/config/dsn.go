package config

import (
	"fmt"
	"net/url"
)

// ExtractDSNComponents derives PGUSER and PGPASSWORD from DATABASE_URL in env
// for any key that appears in needed but is not already set. Passwords that are
// URL-encoded in the DSN (e.g. "p%40ss") are stored decoded in env.
//
// This keeps DATABASE_URL as the single source of truth: callers provide one
// structured value and individual components are extracted on demand.
func ExtractDSNComponents(env map[string]string, needed map[string]struct{}) {
	dsnStr := env["DATABASE_URL"]
	if dsnStr == "" {
		return
	}

	u, err := url.Parse(dsnStr)
	if err != nil {
		return
	}

	if _, req := needed["PGUSER"]; req {
		if _, exists := env["PGUSER"]; !exists {
			env["PGUSER"] = u.User.Username()
		}
	}

	if _, req := needed["PGPASSWORD"]; req {
		if _, exists := env["PGPASSWORD"]; !exists {
			if p, ok := u.User.Password(); ok {
				env["PGPASSWORD"] = p
			}
		}
	}
}

// ExtractKVComponents derives KV_PASSWORD from KV_URL in env for any key that
// appears in needed but is not already set. Passwords that are URL-encoded in
// the URL (e.g. "p%40ss") are stored decoded in env.
//
// This keeps KV_URL as the single source of truth: the app consumes the full
// URL while the Dragonfly container receives only the password as a standalone
// secret file.
func ExtractKVComponents(env map[string]string, needed map[string]struct{}) {
	raw := env["KV_URL"]
	if raw == "" {
		return
	}

	u, err := url.Parse(raw)
	if err != nil {
		return
	}

	if _, req := needed["KV_PASSWORD"]; req {
		if _, exists := env["KV_PASSWORD"]; !exists {
			if p, ok := u.User.Password(); ok {
				env["KV_PASSWORD"] = p
			}
		}
	}
}

// KVConfig holds the parsed connection parameters from a KV_URL.
type KVConfig struct {
	Addr       string
	Password   string
	TLSEnabled bool
}

// ParseKVURL parses a redis:// or rediss:// URL into KV connection parameters.
// An empty raw string returns a config with the default address localhost:6379.
// The rediss:// scheme sets TLSEnabled; the password is URL-decoded automatically.
func ParseKVURL(raw string) (KVConfig, error) {
	if raw == "" {
		return KVConfig{Addr: "localhost:6379"}, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return KVConfig{}, fmt.Errorf("KV_URL: %w", err)
	}
	cfg := KVConfig{Addr: u.Host}
	if cfg.Addr == "" {
		cfg.Addr = "localhost:6379"
	}
	if u.User != nil {
		cfg.Password, _ = u.User.Password()
	}
	cfg.TLSEnabled = u.Scheme == "rediss"
	return cfg, nil
}
