package session

import (
	"context"
	"fmt"
	"time"

	"github.com/go-sum/foundry/pkg/web/cookiecodec"
)

const defaultMaxCookieBytes = 3800

// CookieStore is a Store that encodes session data directly in the client cookie using
// the provided Codec. It requires no server-side storage.
//
// Use cookiecodec.AEAD mode to prevent clients from reading the session payload.
// The "token" exchanged with the middleware is the codec-encoded session blob.
type CookieStore struct {
	codec   *cookiecodec.Codec
	maxSize int
}

// NewCookieStore returns a CookieStore backed by the given Codec.
// Panics if codec is nil.
func NewCookieStore(codec *cookiecodec.Codec) *CookieStore {
	if codec == nil {
		panic("web/session: CookieStore codec must not be nil")
	}
	return &CookieStore{codec: codec, maxSize: defaultMaxCookieBytes}
}

// Read implements Store.
// token is the codec-encoded blob from the client cookie; it is decoded and the raw
// session JSON is returned as data. Returns ErrSessionNotFound on any decode failure.
func (s *CookieStore) Read(_ context.Context, token string) ([]byte, int64, error) {
	if token == "" {
		return nil, 0, ErrSessionNotFound
	}
	data, err := s.codec.Parse(token)
	if err != nil {
		return nil, 0, ErrSessionNotFound
	}
	return []byte(data), 0, nil
}

// Save implements Store.
// It encodes data using the Codec and returns the encoded blob as the new token.
// Returns an error if the encoded size exceeds 3800 bytes.
func (s *CookieStore) Save(_ context.Context, _ string, data []byte, absolute time.Time, _ time.Duration, _ int64) (string, error) {
	token, err := s.codec.Serialize(string(data), absolute)
	if err != nil {
		return "", fmt.Errorf("web/session: cookie store encode: %w", err)
	}
	if len(token) > s.maxSize {
		return "", fmt.Errorf("web/session: encoded cookie payload %d bytes exceeds limit %d", len(token), s.maxSize)
	}
	return token, nil
}

// Delete implements Store. CookieStore has no server-side state; this is a no-op.
func (s *CookieStore) Delete(_ context.Context, _ string) error {
	return nil
}
