-- Auth user queries.
-- These are the persistence operations required by the auth module.
-- Query names and return modes follow sqlc conventions (-- name: X :one/:exec).

-- name: CreateUser :one
INSERT INTO users (email, display_name, role, verified)
VALUES ($1, $2, $3, $4)
RETURNING id, email, display_name, role, verified, webauthn_id, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, email, display_name, role, verified, webauthn_id, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, display_name, role, verified, webauthn_id, created_at, updated_at
FROM users
WHERE email = $1;

-- name: UpdateUserEmail :one
UPDATE users
SET email = $2
WHERE id = $1
RETURNING id, email, display_name, role, verified, webauthn_id, created_at, updated_at;

-- name: SetWebAuthnID :one
UPDATE users SET webauthn_id = $2 WHERE id = $1
RETURNING id, email, display_name, role, verified, webauthn_id, created_at, updated_at;

-- name: SetWebAuthnIDIfNull :one
UPDATE users SET webauthn_id = $2 WHERE id = $1 AND webauthn_id IS NULL
RETURNING id, email, display_name, role, verified, webauthn_id, created_at, updated_at;

-- name: GetUserByWebAuthnID :one
SELECT id, email, display_name, role, verified, webauthn_id, created_at, updated_at
FROM users WHERE webauthn_id = $1;

-- ─── Admin operations ───────────────────────────────────────────────────────

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateUser :one
-- COALESCE(NULLIF(value, ''), column) means: empty string = keep existing value.
-- sqlc.arg() assigns named parameters independent of positional $N numbering.
UPDATE users
SET
    email        = COALESCE(NULLIF(sqlc.arg(email)::text, ''), email),
    display_name = COALESCE(NULLIF(sqlc.arg(display_name)::text, ''), display_name),
    role         = COALESCE(NULLIF(sqlc.arg(role)::text, ''), role)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: HasAdminUser :one
SELECT EXISTS(SELECT 1 FROM users WHERE role = 'admin');

-- ─── WebAuthn credential operations ─────────────────────────────────────────

-- name: CreatePasskeyCredential :one
INSERT INTO webauthn_credentials (
    user_id, credential_id, name, public_key, public_key_alg,
    attestation_type, aaguid, sign_count, clone_warning,
    backup_eligible, backup_state, transports, attachment, last_used_at
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11, $12, $13, $14
)
RETURNING *;

-- name: GetPasskeyCredentialByCredentialID :one
SELECT * FROM webauthn_credentials
WHERE credential_id = $1;

-- name: GetPasskeyCredentialByIDForUser :one
SELECT * FROM webauthn_credentials
WHERE id = $1 AND user_id = $2;

-- name: ListPasskeyCredentialsByUserID :many
SELECT * FROM webauthn_credentials
WHERE user_id = $1
ORDER BY created_at DESC, id DESC;

-- name: TouchPasskeyCredential :exec
UPDATE webauthn_credentials
SET sign_count    = GREATEST(sign_count, $2),
    clone_warning = clone_warning OR $3,
    last_used_at  = $4,
    updated_at    = NOW()
WHERE id = $1;

-- name: RenamePasskeyCredential :one
UPDATE webauthn_credentials
SET name = $3
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeletePasskeyCredential :one
DELETE FROM webauthn_credentials
WHERE id = $1 AND user_id = $2
RETURNING id;
