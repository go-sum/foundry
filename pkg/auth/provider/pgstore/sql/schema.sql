-- Depends on auth schema (users table must exist).

-- OAuth 2.0 Clients
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

-- Authorization Codes (single-use, short-lived)
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

CREATE INDEX IF NOT EXISTS idx_oauth_codes_expires_at
    ON oauth_authorization_codes (expires_at);

-- OAuth Tokens (access + refresh)
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

CREATE INDEX IF NOT EXISTS idx_oauth_tokens_user_client
    ON oauth_tokens (user_id, client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_tokens_expires_at
    ON oauth_tokens (expires_at) WHERE NOT revoked;

-- User Consent Records
CREATE TABLE IF NOT EXISTS oauth_consents (
    id          UUID         PRIMARY KEY DEFAULT uuidv7(),
    user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id   VARCHAR(255) NOT NULL REFERENCES oauth_clients(client_id) ON DELETE CASCADE,
    scopes      TEXT[]       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, client_id)
);

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
