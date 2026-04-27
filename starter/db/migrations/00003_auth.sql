-- Auto-generated migration - do not edit

-- +migrate Up
CREATE TABLE IF NOT EXISTS users (
    id            UUID         PRIMARY KEY DEFAULT uuidv7(),
    email         CITEXT       NOT NULL UNIQUE,
    display_name  VARCHAR(255) NOT NULL,
    role          VARCHAR(50)  NOT NULL DEFAULT 'user',
    verified      BOOLEAN      NOT NULL DEFAULT false,
    webauthn_id   BYTEA        UNIQUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id               UUID         PRIMARY KEY DEFAULT uuidv7(),
    user_id          UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id    BYTEA        NOT NULL UNIQUE,
    name             VARCHAR(255) NOT NULL DEFAULT '',
    public_key       BYTEA        NOT NULL,
    public_key_alg   BIGINT       NOT NULL,
    attestation_type VARCHAR(255) NOT NULL DEFAULT '',
    aaguid           BYTEA        NOT NULL,
    sign_count       BIGINT       NOT NULL DEFAULT 0,
    clone_warning    BOOLEAN      NOT NULL DEFAULT false,
    backup_eligible  BOOLEAN      NOT NULL DEFAULT false,
    backup_state     BOOLEAN      NOT NULL DEFAULT false,
    transports       TEXT[]       NOT NULL DEFAULT '{}',
    attachment       VARCHAR(64)  NOT NULL DEFAULT '',
    last_used_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);

CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user_created ON webauthn_credentials (user_id, created_at DESC);

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'users_updated_at'
    ) THEN
        CREATE TRIGGER users_updated_at
            BEFORE UPDATE ON users
            FOR EACH ROW
            EXECUTE FUNCTION update_updated_at();
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'webauthn_credentials_updated_at'
    ) THEN
        CREATE TRIGGER webauthn_credentials_updated_at
            BEFORE UPDATE ON webauthn_credentials
            FOR EACH ROW
            EXECUTE FUNCTION update_updated_at();
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS oauth_clients (
    id             UUID         PRIMARY KEY DEFAULT uuidv7(),
    client_id      VARCHAR(255) NOT NULL UNIQUE,
    client_secret  VARCHAR(255) NOT NULL DEFAULT '',
    name           VARCHAR(255) NOT NULL,
    redirect_uris  TEXT[]       NOT NULL DEFAULT '{}',
    scopes         TEXT[]       NOT NULL DEFAULT '{}',
    public         BOOLEAN      NOT NULL DEFAULT false,
    first_party    BOOLEAN      NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS oauth_authorization_codes (
    code            VARCHAR(255) PRIMARY KEY,
    client_id       VARCHAR(255) NOT NULL REFERENCES oauth_clients(client_id) ON DELETE CASCADE,
    user_id         UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    redirect_uri    TEXT         NOT NULL,
    scopes          TEXT[]       NOT NULL DEFAULT '{}',
    code_challenge  VARCHAR(255) NOT NULL DEFAULT '',
    nonce           VARCHAR(255) NOT NULL DEFAULT '',
    used            BOOLEAN      NOT NULL DEFAULT false,
    expires_at      TIMESTAMPTZ  NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS oauth_tokens (
    id          UUID         PRIMARY KEY DEFAULT uuidv7(),
    token_hash  VARCHAR(64)  NOT NULL UNIQUE,
    token_type  VARCHAR(20)  NOT NULL,
    client_id   VARCHAR(255) NOT NULL REFERENCES oauth_clients(client_id) ON DELETE CASCADE,
    user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scopes      TEXT[]       NOT NULL DEFAULT '{}',
    revoked     BOOLEAN      NOT NULL DEFAULT false,
    parent_id   UUID         REFERENCES oauth_tokens(id) ON DELETE SET NULL,
    expires_at  TIMESTAMPTZ  NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS oauth_consents (
    id          UUID         PRIMARY KEY DEFAULT uuidv7(),
    user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id   VARCHAR(255) NOT NULL REFERENCES oauth_clients(client_id) ON DELETE CASCADE,
    scopes      TEXT[]       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oauth_codes_expires_at ON oauth_authorization_codes (expires_at);

CREATE INDEX IF NOT EXISTS idx_oauth_tokens_user_client ON oauth_tokens (user_id, client_id);

CREATE INDEX IF NOT EXISTS idx_oauth_tokens_expires_at ON oauth_tokens (expires_at) WHERE NOT revoked;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'oauth_clients_updated_at') THEN
        CREATE TRIGGER oauth_clients_updated_at
            BEFORE UPDATE ON oauth_clients FOR EACH ROW EXECUTE FUNCTION update_updated_at();
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'oauth_consents_updated_at') THEN
        CREATE TRIGGER oauth_consents_updated_at
            BEFORE UPDATE ON oauth_consents FOR EACH ROW EXECUTE FUNCTION update_updated_at();
    END IF;
END $$;

-- +migrate Down
DROP TABLE IF EXISTS users CASCADE;

DROP TABLE IF EXISTS webauthn_credentials CASCADE;

DROP INDEX IF EXISTS idx_users_role;

DROP INDEX IF EXISTS idx_webauthn_credentials_user_created;

DROP TRIGGER IF EXISTS users_updated_at ON users;

DROP TRIGGER IF EXISTS webauthn_credentials_updated_at ON webauthn_credentials;

DROP TABLE IF EXISTS oauth_clients CASCADE;

DROP TABLE IF EXISTS oauth_authorization_codes CASCADE;

DROP TABLE IF EXISTS oauth_tokens CASCADE;

DROP TABLE IF EXISTS oauth_consents CASCADE;

DROP INDEX IF EXISTS idx_oauth_codes_expires_at;

DROP INDEX IF EXISTS idx_oauth_tokens_user_client;

DROP INDEX IF EXISTS idx_oauth_tokens_expires_at;

DROP TRIGGER IF EXISTS oauth_clients_updated_at ON oauth_clients;

DROP TRIGGER IF EXISTS oauth_consents_updated_at ON oauth_consents;
