package auth

import (
	"context"
	"errors"
	"time"

	websession "github.com/go-sum/foundry/pkg/web/session"
)

// DefaultTransactionTTL is the maximum age of an OAuthTransaction before it is
// considered expired.
const DefaultTransactionTTL = 10 * time.Minute

// ErrTransactionExpired is returned by CompleteAuth when the transaction CreatedAt
// timestamp is non-zero and older than DefaultTransactionTTL.
var ErrTransactionExpired = errors.New("oauth: transaction expired")

// CompleteAuth finalizes an OAuth flow. It:
//  1. Rejects the transaction if it is older than DefaultTransactionTTL.
//  2. Calls sess.Regenerate() to prevent session fixation after authentication.
//  3. Removes the transaction from the session via sess.Unset(SessionKey).
//  4. Returns the sanitized return-to URL from the transaction.
//
// The caller is responsible for verifying state, nonce, and exchanging the code.
func CompleteAuth(ctx context.Context, sess *websession.Session, tx OAuthTransaction) (returnTo string, err error) {
	if !tx.CreatedAt.IsZero() && time.Since(tx.CreatedAt) > DefaultTransactionTTL {
		return "", ErrTransactionExpired
	}
	sess.Regenerate()
	sess.Unset(SessionKey)
	return tx.ReturnTo, nil
}
