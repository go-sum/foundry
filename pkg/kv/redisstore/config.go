package redisstore

import (
	"crypto/tls"
	"time"
)

// Config holds connection parameters for a Redis-protocol server.
type Config struct {
	Addr         string        `validate:"required"      help:"host:port of the Redis-protocol server"`
	Password     string        `                         help:"authentication password (empty for no auth)"`
	DB           int           `validate:"min=0,max=15"  help:"database number (default 0)"`
	PoolSize     int           `validate:"min=0"         help:"maximum connections (0 uses default of 10)"`
	MinIdleConns int           `validate:"min=0"         help:"minimum idle connections"`
	DialTimeout  time.Duration `                         help:"connection dial timeout (default 5s)"`
	ReadTimeout  time.Duration `                         help:"read timeout (default 3s)"`
	WriteTimeout time.Duration `                         help:"write timeout (default 3s)"`
	TLSConfig    *tls.Config   `                         help:"optional TLS config for managed Redis services"`
}

// InitialRedisConfig returns a Config with sane defaults for local development.
func InitialRedisConfig() Config {
	return Config{
		Addr:         "localhost:6379",
		PoolSize:     10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}
