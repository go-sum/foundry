package cookiecodec

import (
	"errors"
	"fmt"
	"time"
)

// Mode selects the security mode of the codec.
type Mode int

const (
	// Signed uses HMAC-SHA256. The cookie value is visible to the client.
	Signed Mode = iota
	// AEAD uses XChaCha20-Poly1305. The cookie value is encrypted.
	AEAD
)

// Config configures a Codec.
type Config struct {
	// Name is the cookie name. It is bound into every MAC/AAD so a cookie
	// value signed for one name cannot be reused for another.
	Name string
	// Secrets is the list of signing/encryption keys. At least one is required.
	// Secrets[0] is used for new values; all are tried during verification.
	// Rotate by prepending a new secret.
	Secrets [][]byte `validate:"required,min=1"`
	// Mode selects Signed (HMAC-only) or AEAD (encrypted) mode.
	Mode Mode
}

// Codec serializes and parses tamper-evident cookie values.
type Codec struct {
	cfg Config
}

var (
	// ErrInvalid is returned when a cookie value fails authentication.
	ErrInvalid = errors.New("cookiecodec: invalid cookie value")

	// ErrExpired is returned when a cookie value is past its expiry.
	ErrExpired = errors.New("cookiecodec: cookie value expired")

	// ErrEmptySecrets is returned when Config.Secrets is empty.
	ErrEmptySecrets = errors.New("cookiecodec: Secrets must not be empty")

	// ErrInvalidMode is returned when Config.Mode is not Signed or AEAD.
	ErrInvalidMode = errors.New("cookiecodec: Mode must be Signed or AEAD")

	// ErrEmptyName is returned when Config.Name is empty.
	ErrEmptyName = errors.New("cookiecodec: Name must not be empty")
)

// New returns a new Codec. Returns ErrEmptySecrets if cfg.Secrets is empty
// or if any secret is empty. Returns ErrInvalidMode for unrecognised Mode values.
func New(cfg Config) (*Codec, error) {
	if len(cfg.Secrets) == 0 {
		return nil, ErrEmptySecrets
	}
	for i, s := range cfg.Secrets {
		if len(s) == 0 {
			return nil, fmt.Errorf("cookiecodec: Secrets[%d] must not be empty", i)
		}
	}
	if cfg.Name == "" {
		return nil, ErrEmptyName
	}
	if cfg.Mode != Signed && cfg.Mode != AEAD {
		return nil, ErrInvalidMode
	}
	// Deep-copy secrets so post-construction mutation of the caller's slice
	// cannot affect the active keys.
	secrets := make([][]byte, len(cfg.Secrets))
	for i, s := range cfg.Secrets {
		cp := make([]byte, len(s))
		copy(cp, s)
		secrets[i] = cp
	}
	cfg.Secrets = secrets
	return &Codec{cfg: cfg}, nil
}

// Serialize encodes and signs/encrypts value. exp is the absolute expiry time;
// use the zero value for a session cookie (no server-side expiry enforcement).
// Returns a base64url-encoded string suitable as a Set-Cookie value.
func (c *Codec) Serialize(value string, exp time.Time) (string, error) {
	switch c.cfg.Mode {
	case AEAD:
		return serializeAEAD(c.cfg.Name, c.cfg.Secrets[0], value, exp)
	case Signed:
		return serializeSigned(c.cfg.Name, c.cfg.Secrets[0], value, exp)
	default:
		return "", ErrInvalidMode
	}
}

// Parse decodes and verifies a base64url-encoded cookie value produced by Serialize.
// Returns ErrInvalid if verification fails, ErrExpired if the cookie is past its expiry.
func (c *Codec) Parse(encoded string) (string, error) {
	switch c.cfg.Mode {
	case AEAD:
		return parseAEAD(c.cfg.Name, c.cfg.Secrets, encoded)
	case Signed:
		return parseSigned(c.cfg.Name, c.cfg.Secrets, encoded)
	default:
		return "", ErrInvalidMode
	}
}
