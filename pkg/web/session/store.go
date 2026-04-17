package session

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrSessionNotFound is returned by Store.Read when the token is unknown, expired, or invalid.
	ErrSessionNotFound = errors.New("web/session: session not found")

	// ErrVersionConflict is returned by Store.Save when the record was updated since Read.
	ErrVersionConflict = errors.New("web/session: version conflict")
)

// Store persists and retrieves session data by an opaque token.
//
// Implementations are responsible for encoding the token written to the client cookie:
//   - MemoryStore uses a random 256-bit session ID; the token is that ID.
//   - CookieStore uses an AEAD-encrypted blob; the token is the encoded blob.
type Store interface {
	// Read loads session data for the given token.
	// Returns ErrSessionNotFound if the token is unknown, expired, or fails verification.
	Read(ctx context.Context, token string) (data []byte, version int64, err error)

	// Save persists data and returns the token to write into the client cookie.
	// Pass the version from Read for existing sessions; pass 0 for new sessions.
	// Implementations may enforce optimistic concurrency and return ErrVersionConflict.
	Save(ctx context.Context, token string, data []byte, absolute time.Time, idleTTL time.Duration, version int64) (newToken string, err error)

	// Delete removes session data. It is a no-op if the token is not found.
	Delete(ctx context.Context, token string) error
}
