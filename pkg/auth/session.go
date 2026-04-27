package auth

import (
	"github.com/go-sum/foundry/pkg/web/session"
	"github.com/google/uuid"
)

const (
	sessionKeyUserID          = "auth.user_id"
	sessionKeyDisplayName     = "auth.display_name"
	sessionKeyVerified        = "auth.verified"
	sessionKeyPendingFlow     = "auth.pending_flow"
	sessionKeyPasskeyCeremony = "auth.passkey_ceremony"
)

// SetAuth records the authenticated user in the session. It is exported so
// OAuth callback handlers can establish the same session state after resolving
// identity from the userinfo endpoint.
func SetAuth(sess *session.Session, userID, displayName string, verified bool) error {
	if err := sess.Set(sessionKeyUserID, userID); err != nil {
		return err
	}
	if err := sess.Set(sessionKeyDisplayName, displayName); err != nil {
		return err
	}
	if err := sess.Set(sessionKeyVerified, verified); err != nil {
		return err
	}
	sess.Unset(sessionKeyPendingFlow)
	return nil
}

func getUserID(sess *session.Session) (string, bool) {
	id, ok, _ := session.Get[string](sess, sessionKeyUserID)
	return id, ok && id != ""
}

// IsAuthenticated reports whether the session holds a valid authenticated user.
func IsAuthenticated(sess *session.Session) bool {
	_, ok := getUserID(sess)
	return ok
}

func getDisplayName(sess *session.Session) (string, bool) {
	name, ok, _ := session.Get[string](sess, sessionKeyDisplayName)
	return name, ok && name != ""
}

func getVerified(sess *session.Session) bool {
	v, _, _ := session.Get[bool](sess, sessionKeyVerified)
	return v
}

func setPendingFlow(sess *session.Session, flow PendingFlow) error {
	return sess.Set(sessionKeyPendingFlow, flow)
}

func getPendingFlow(sess *session.Session) (PendingFlow, bool) {
	flow, ok, _ := session.Get[PendingFlow](sess, sessionKeyPendingFlow)
	return flow, ok
}

type passkeyCeremonyState struct {
	Operation string          `json:"operation"`
	Ceremony  PasskeyCeremony `json:"ceremony"`
	UserID    uuid.UUID       `json:"user_id,omitempty"`
}

func setPasskeyCeremony(sess *session.Session, state passkeyCeremonyState) error {
	return sess.Set(sessionKeyPasskeyCeremony, state)
}

func getPasskeyCeremony(sess *session.Session) (passkeyCeremonyState, bool) {
	state, ok, _ := session.Get[passkeyCeremonyState](sess, sessionKeyPasskeyCeremony)
	return state, ok
}

func clearPasskeyCeremony(sess *session.Session) {
	sess.Unset(sessionKeyPasskeyCeremony)
}
