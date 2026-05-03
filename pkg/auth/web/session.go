package authweb

import (
	"github.com/go-sum/foundry/pkg/auth"
	"github.com/go-sum/foundry/pkg/web/session"
	"github.com/google/uuid"
)

const (
	sessionKeyPendingFlow     = "auth.pending_flow"
	sessionKeyPasskeyCeremony = "auth.passkey_ceremony"
)

func setPendingFlow(sess *session.Session, flow auth.PendingFlow) error {
	return sess.Set(sessionKeyPendingFlow, flow)
}

func getPendingFlow(sess *session.Session) (auth.PendingFlow, bool) {
	flow, ok, _ := session.Get[auth.PendingFlow](sess, sessionKeyPendingFlow)
	return flow, ok
}

type passkeyCeremonyState struct {
	Operation string               `json:"operation"`
	Ceremony  auth.PasskeyCeremony `json:"ceremony"`
	UserID    uuid.UUID            `json:"user_id,omitempty"`
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
