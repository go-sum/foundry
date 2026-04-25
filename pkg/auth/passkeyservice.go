package auth

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

// CredentialStore defines persistence operations for WebAuthn credentials.
type CredentialStore interface {
	CreateCredential(ctx context.Context, cred PasskeyCredential) (PasskeyCredential, error)
	GetByCredentialID(ctx context.Context, credentialID []byte) (PasskeyCredential, error)
	GetByIDForUser(ctx context.Context, userID, id uuid.UUID) (PasskeyCredential, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]PasskeyCredential, error)
	TouchPasskeyCredential(ctx context.Context, id uuid.UUID, signCount int64, cloneWarning bool, lastUsed time.Time) error
	RenameCredential(ctx context.Context, id, userID uuid.UUID, name string) (PasskeyCredential, error)
	DeleteCredential(ctx context.Context, id, userID uuid.UUID) error
}

// PasskeyService implements WebAuthn passkey authentication using the go-webauthn library.
type PasskeyService struct {
	webAuthn    *webauthn.WebAuthn
	users       UserWriter
	credentials CredentialStore
	clock       func() time.Time
}

// webauthnUser adapts a User and its credentials to the webauthn.User interface.
type webauthnUser struct {
	id          []byte
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u webauthnUser) WebAuthnID() []byte                         { return u.id }
func (u webauthnUser) WebAuthnName() string                       { return u.name }
func (u webauthnUser) WebAuthnDisplayName() string                { return u.displayName }
func (u webauthnUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// NewPasskeyService constructs a PasskeyService. Returns an error if the
// WebAuthn configuration is invalid.
func NewPasskeyService(users UserWriter, credentials CredentialStore, cfg PasskeyConfig) (*PasskeyService, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("NewPasskeyService: %w", err)
	}

	waCfg := &webauthn.Config{
		RPDisplayName:         cfg.RPDisplayName,
		RPID:                  cfg.RPID,
		RPOrigins:             cfg.RPOrigins,
		AttestationPreference: protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirement(cfg.ResidentKey),
			UserVerification: protocol.UserVerificationRequirement(cfg.UserVerification),
		},
	}

	// Apply server-side timeout enforcement only when a non-zero duration is configured.
	if cfg.RegistrationTimeout > 0 || cfg.AuthenticationTimeout > 0 {
		waCfg.Timeouts = webauthn.TimeoutsConfig{
			Login: webauthn.TimeoutConfig{
				Enforce:    cfg.AuthenticationTimeout > 0,
				Timeout:    cfg.AuthenticationTimeout,
				TimeoutUVD: cfg.AuthenticationTimeout,
			},
			Registration: webauthn.TimeoutConfig{
				Enforce:    cfg.RegistrationTimeout > 0,
				Timeout:    cfg.RegistrationTimeout,
				TimeoutUVD: cfg.RegistrationTimeout,
			},
		}
	}

	wa, err := webauthn.New(waCfg)
	if err != nil {
		return nil, fmt.Errorf("NewPasskeyService: configure webauthn: %w", err)
	}
	return &PasskeyService{
		webAuthn:    wa,
		users:       users,
		credentials: credentials,
		clock:       func() time.Time { return time.Now().UTC() },
	}, nil
}

// defaultCredentialParameters restricts the advertised algorithms to ES256 and RS256,
// matching the working reference implementation and avoiding exotic algs that
// some authenticators or browsers may reject.
var defaultCredentialParameters = []protocol.CredentialParameter{
	{Type: protocol.PublicKeyCredentialType, Algorithm: webauthncose.AlgES256},
	{Type: protocol.PublicKeyCredentialType, Algorithm: webauthncose.AlgRS256},
}

// BeginRegistration starts a passkey registration ceremony for the given user.
// If the user has no WebAuthn ID yet, one is generated and persisted.
// Returns the options to send to the client and the ceremony state to store server-side.
func (s *PasskeyService) BeginRegistration(ctx context.Context, userID uuid.UUID) (PasskeyCreationOptions, PasskeyCeremony, error) {
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return PasskeyCreationOptions{}, PasskeyCeremony{}, fmt.Errorf("PasskeyService.BeginRegistration: load user: %w", err)
	}

	if len(user.WebAuthnID) == 0 {
		handle := make([]byte, 64)
		if _, err := rand.Read(handle); err != nil {
			return PasskeyCreationOptions{}, PasskeyCeremony{}, fmt.Errorf("PasskeyService.BeginRegistration: generate handle: %w", err)
		}
		// Use conditional update to avoid overwriting an existing handle set by a concurrent request.
		updated, err := s.users.SetWebAuthnIDIfNull(ctx, userID, handle)
		if errors.Is(err, ErrWebAuthnIDAlreadySet) {
			// Another goroutine already set it; re-read the current value.
			user, err = s.users.GetUserByID(ctx, userID)
			if err != nil {
				return PasskeyCreationOptions{}, PasskeyCeremony{}, fmt.Errorf("PasskeyService.BeginRegistration: reload user after concurrent handle set: %w", err)
			}
		} else if err != nil {
			return PasskeyCreationOptions{}, PasskeyCeremony{}, fmt.Errorf("PasskeyService.BeginRegistration: persist handle: %w", err)
		} else {
			user = updated
		}
	}

	existing, err := s.credentials.ListByUserID(ctx, userID)
	if err != nil {
		return PasskeyCreationOptions{}, PasskeyCeremony{}, fmt.Errorf("PasskeyService.BeginRegistration: list credentials: %w", err)
	}

	waCreds := toWebAuthnCredentials(existing)
	adapter := webauthnUser{
		id:          user.WebAuthnID,
		name:        user.Email,
		displayName: user.DisplayName,
		credentials: waCreds,
	}

	opts := []webauthn.RegistrationOption{
		webauthn.WithCredentialParameters(defaultCredentialParameters),
	}
	if len(waCreds) > 0 {
		descs := webauthn.Credentials(waCreds).CredentialDescriptors()
		opts = append(opts, webauthn.WithExclusions(descs))
	}

	creation, sessionData, err := s.webAuthn.BeginRegistration(adapter, opts...)
	if err != nil {
		return PasskeyCreationOptions{}, PasskeyCeremony{}, fmt.Errorf("PasskeyService.BeginRegistration: %w", err)
	}

	raw, err := json.Marshal(creation)
	if err != nil {
		return PasskeyCreationOptions{}, PasskeyCeremony{}, fmt.Errorf("PasskeyService.BeginRegistration: marshal options: %w", err)
	}
	// creation JSON is {"publicKey": {...}} -- extract the inner publicKey value
	var wrapper struct {
		PublicKey json.RawMessage `json:"publicKey"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return PasskeyCreationOptions{}, PasskeyCeremony{}, fmt.Errorf("PasskeyService.BeginRegistration: unwrap options: %w", err)
	}
	return PasskeyCreationOptions{PublicKey: wrapper.PublicKey}, toCeremony(sessionData), nil
}

// FinishRegistration completes a passkey registration ceremony and persists the new credential.
func (s *PasskeyService) FinishRegistration(ctx context.Context, userID uuid.UUID, name string, ceremony PasskeyCeremony, r *http.Request) (PasskeyCredential, error) {
	if !ceremony.Expires.IsZero() && s.clock().After(ceremony.Expires) {
		return PasskeyCredential{}, fmt.Errorf("PasskeyService.FinishRegistration: %w", ErrPasskeyVerificationFailed)
	}

	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return PasskeyCredential{}, fmt.Errorf("PasskeyService.FinishRegistration: load user: %w", err)
	}

	existing, err := s.credentials.ListByUserID(ctx, userID)
	if err != nil {
		return PasskeyCredential{}, fmt.Errorf("PasskeyService.FinishRegistration: list credentials: %w", err)
	}

	adapter := webauthnUser{
		id:          user.WebAuthnID,
		name:        user.Email,
		displayName: user.DisplayName,
		credentials: toWebAuthnCredentials(existing),
	}

	sd := fromCeremony(ceremony)
	credential, err := s.webAuthn.FinishRegistration(adapter, sd, r)
	if err != nil {
		var protoErr *protocol.Error
		if errors.As(err, &protoErr) {
			return PasskeyCredential{}, fmt.Errorf("PasskeyService.FinishRegistration: %w: %w", classifyProtocolError(ctx, protoErr), err)
		}
		return PasskeyCredential{}, fmt.Errorf("PasskeyService.FinishRegistration: %w", err)
	}

	transports := make([]string, len(credential.Transport))
	for i, t := range credential.Transport {
		transports[i] = string(t)
	}

	cred := PasskeyCredential{
		UserID:          userID,
		CredentialID:    credential.ID,
		Name:            name,
		PublicKey:       credential.PublicKey,
		PublicKeyAlg:    credential.Attestation.PublicKeyAlgorithm,
		AttestationType: credential.AttestationType,
		AAGUID:          credential.Authenticator.AAGUID,
		SignCount:       int64(credential.Authenticator.SignCount),
		CloneWarning:    credential.Authenticator.CloneWarning,
		BackupEligible:  credential.Flags.BackupEligible,
		BackupState:     credential.Flags.BackupState,
		Transports:      transports,
		Attachment:      string(credential.Authenticator.Attachment),
	}

	created, err := s.credentials.CreateCredential(ctx, cred)
	if err != nil {
		return PasskeyCredential{}, fmt.Errorf("PasskeyService.FinishRegistration: persist credential: %w", err)
	}
	return created, nil
}

// BeginAuthentication starts a discoverable-login ceremony.
// Returns the assertion options for the client and ceremony state to store server-side.
func (s *PasskeyService) BeginAuthentication(ctx context.Context) (PasskeyRequestOptions, PasskeyCeremony, error) {
	assertion, sessionData, err := s.webAuthn.BeginDiscoverableLogin()
	if err != nil {
		return PasskeyRequestOptions{}, PasskeyCeremony{}, fmt.Errorf("PasskeyService.BeginAuthentication: %w", err)
	}

	raw, err := json.Marshal(assertion)
	if err != nil {
		return PasskeyRequestOptions{}, PasskeyCeremony{}, fmt.Errorf("PasskeyService.BeginAuthentication: marshal options: %w", err)
	}
	var wrapper struct {
		PublicKey json.RawMessage `json:"publicKey"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return PasskeyRequestOptions{}, PasskeyCeremony{}, fmt.Errorf("PasskeyService.BeginAuthentication: unwrap options: %w", err)
	}
	return PasskeyRequestOptions{PublicKey: wrapper.PublicKey}, toCeremony(sessionData), nil
}

// FinishAuthentication completes the discoverable login ceremony and returns the verified user.
func (s *PasskeyService) FinishAuthentication(ctx context.Context, ceremony PasskeyCeremony, r *http.Request) (VerifyResult, error) {
	if !ceremony.Expires.IsZero() && s.clock().After(ceremony.Expires) {
		return VerifyResult{}, fmt.Errorf("PasskeyService.FinishAuthentication: %w", ErrPasskeyVerificationFailed)
	}

	var verifiedUser User
	var credentialID []byte

	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		user, err := s.users.GetUserByWebAuthnID(ctx, userHandle)
		if err != nil {
			return nil, fmt.Errorf("lookup user by webauthn handle: %w", err)
		}
		verifiedUser = user
		credentialID = rawID

		existing, err := s.credentials.ListByUserID(ctx, user.ID)
		if err != nil {
			return nil, fmt.Errorf("list credentials: %w", err)
		}

		return webauthnUser{
			id:          user.WebAuthnID,
			name:        user.Email,
			displayName: user.DisplayName,
			credentials: toWebAuthnCredentials(existing),
		}, nil
	}

	sd := fromCeremony(ceremony)
	credential, err := s.webAuthn.FinishDiscoverableLogin(handler, sd, r)
	if err != nil {
		var protoErr *protocol.Error
		if errors.As(err, &protoErr) {
			return VerifyResult{}, fmt.Errorf("PasskeyService.FinishAuthentication: %w: %w", classifyProtocolError(ctx, protoErr), err)
		}
		return VerifyResult{}, fmt.Errorf("PasskeyService.FinishAuthentication: %w", err)
	}

	// Enforce the same verified-user invariant as the email-TOTP path.
	// An unverified user must not be admitted regardless of how their passkey row
	// came to exist (admin creation, migration, test fixture, etc.).
	if !verifiedUser.Verified {
		slog.WarnContext(ctx, "passkey auth rejected: unverified user",
			"user_id", verifiedUser.ID,
		)
		return VerifyResult{}, fmt.Errorf(
			"PasskeyService.FinishAuthentication: %w", ErrInvalidCredentials,
		)
	}

	// Enforce clone warning before updating counters.
	if credential.Authenticator.CloneWarning {
		stored, err := s.credentials.GetByCredentialID(ctx, credentialID)
		if err != nil {
			return VerifyResult{}, fmt.Errorf("PasskeyService.FinishAuthentication: lookup credential for clone check: %w", err)
		}
		slog.WarnContext(ctx, "passkey clone warning detected",
			"credential_id", stored.ID,
			"user_id", verifiedUser.ID,
			"stored_sign_count", stored.SignCount,
			"received_sign_count", credential.Authenticator.SignCount,
		)
		return VerifyResult{}, fmt.Errorf("PasskeyService.FinishAuthentication: %w", ErrPasskeyCloneDetected)
	}

	// Look up the stored credential to get its UUID for updating counters.
	stored, err := s.credentials.GetByCredentialID(ctx, credentialID)
	if err != nil {
		return VerifyResult{}, fmt.Errorf("PasskeyService.FinishAuthentication: lookup credential: %w", err)
	}

	if err := s.credentials.TouchPasskeyCredential(ctx, stored.ID, int64(credential.Authenticator.SignCount), credential.Authenticator.CloneWarning, s.clock()); err != nil {
		return VerifyResult{}, fmt.Errorf("PasskeyService.FinishAuthentication: update credential: %w", err)
	}

	return VerifyResult{
		User:   verifiedUser,
		Method: string(MethodPasskey),
	}, nil
}

// GetPasskey returns a single passkey credential by ID, scoped to the given user.
// Returns ErrPasskeyNotFound if the credential does not exist or does not belong to the user.
func (s *PasskeyService) GetPasskey(ctx context.Context, userID, passkeyID uuid.UUID) (PasskeyCredential, error) {
	cred, err := s.credentials.GetByIDForUser(ctx, userID, passkeyID)
	if err != nil {
		return PasskeyCredential{}, fmt.Errorf("PasskeyService.GetPasskey: %w", err)
	}
	return cred, nil
}

// ListPasskeys returns all registered passkeys for the given user.
func (s *PasskeyService) ListPasskeys(ctx context.Context, userID uuid.UUID) ([]PasskeyCredential, error) {
	creds, err := s.credentials.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("PasskeyService.ListPasskeys: %w", err)
	}
	return creds, nil
}

// DeletePasskey removes a passkey credential. Enforces ownership of the credential.
func (s *PasskeyService) DeletePasskey(ctx context.Context, userID, passkeyID uuid.UUID) error {
	if err := s.credentials.DeleteCredential(ctx, passkeyID, userID); err != nil {
		return fmt.Errorf("PasskeyService.DeletePasskey: %w", err)
	}
	return nil
}

// RenamePasskey changes the display name of a passkey credential.
func (s *PasskeyService) RenamePasskey(ctx context.Context, userID, passkeyID uuid.UUID, name string) (PasskeyCredential, error) {
	cred, err := s.credentials.RenameCredential(ctx, passkeyID, userID, name)
	if err != nil {
		return PasskeyCredential{}, fmt.Errorf("PasskeyService.RenamePasskey: %w", err)
	}
	return cred, nil
}

// toCeremony converts a go-webauthn SessionData to the neutral PasskeyCeremony type.
func toCeremony(sd *webauthn.SessionData) PasskeyCeremony {
	params := make([]PasskeyCredentialParameter, len(sd.CredParams))
	for i, p := range sd.CredParams {
		params[i] = PasskeyCredentialParameter{
			Type:      string(p.Type),
			Algorithm: int64(p.Algorithm),
		}
	}
	return PasskeyCeremony{
		Challenge:            []byte(sd.Challenge),
		RelyingPartyID:       sd.RelyingPartyID,
		UserID:               sd.UserID,
		AllowedCredentialIDs: sd.AllowedCredentialIDs,
		UserVerification:     string(sd.UserVerification),
		Mediation:            string(sd.Mediation),
		Extensions:           sd.Extensions,
		Expires:              sd.Expires,
		CredentialParameters: params,
	}
}

// fromCeremony converts a neutral PasskeyCeremony back to go-webauthn SessionData.
func fromCeremony(c PasskeyCeremony) webauthn.SessionData {
	params := make([]protocol.CredentialParameter, len(c.CredentialParameters))
	for i, p := range c.CredentialParameters {
		params[i] = protocol.CredentialParameter{
			Type:      protocol.CredentialType(p.Type),
			Algorithm: webauthncose.COSEAlgorithmIdentifier(p.Algorithm),
		}
	}
	return webauthn.SessionData{
		Challenge:            string(c.Challenge),
		RelyingPartyID:       c.RelyingPartyID,
		UserID:               c.UserID,
		AllowedCredentialIDs: c.AllowedCredentialIDs,
		UserVerification:     protocol.UserVerificationRequirement(c.UserVerification),
		Mediation:            protocol.CredentialMediationRequirement(c.Mediation),
		Extensions:           c.Extensions,
		Expires:              c.Expires,
		CredParams:           params,
	}
}

// classifyProtocolError distinguishes server-state protocol errors (bugs) from
// user-facing verification failures. Server-state errors are logged at error level.
func classifyProtocolError(ctx context.Context, protoErr *protocol.Error) error {
	devInfo := protoErr.DevInfo
	if strings.Contains(devInfo, "ID mismatch for User and Session") ||
		strings.Contains(devInfo, "Session has Expired") {
		slog.ErrorContext(ctx, "passkey server state error", "dev_info", devInfo, "type", protoErr.Type)
		return ErrPasskeyServerState
	}
	return ErrPasskeyVerificationFailed
}

// toWebAuthnCredentials converts domain credentials to the go-webauthn library type
// for use in exclusion lists and user adapters.
func toWebAuthnCredentials(creds []PasskeyCredential) []webauthn.Credential {
	result := make([]webauthn.Credential, len(creds))
	for i, c := range creds {
		transports := make([]protocol.AuthenticatorTransport, len(c.Transports))
		for j, t := range c.Transports {
			transports[j] = protocol.AuthenticatorTransport(t)
		}
		result[i] = webauthn.Credential{
			ID:              c.CredentialID,
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Transport:       transports,
			Flags: webauthn.CredentialFlags{
				BackupEligible: c.BackupEligible,
				BackupState:    c.BackupState,
			},
			Authenticator: webauthn.Authenticator{
				AAGUID:       c.AAGUID,
				SignCount:    uint32(c.SignCount),
				CloneWarning: c.CloneWarning,
				Attachment:   protocol.AuthenticatorAttachment(c.Attachment),
			},
		}
	}
	return result
}
