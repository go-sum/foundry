package auth

import (
	"context"

	websession "github.com/go-sum/web/session"
)

// CompleteAuth finalizes an OAuth flow. It:
//  1. Calls sess.Regenerate() to prevent session fixation after authentication.
//  2. Returns the sanitized return-to URL from the transaction.
//
// The caller is responsible for verifying state, nonce, and exchanging the code.
func CompleteAuth(ctx context.Context, sess *websession.Session, tx OAuthTransaction) (returnTo string, err error) {
	sess.Regenerate()
	return tx.ReturnTo, nil
}
