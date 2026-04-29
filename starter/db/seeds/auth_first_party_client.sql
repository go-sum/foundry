INSERT INTO oauth_clients (
    client_id, client_secret, name, redirect_uris, scopes, public, first_party
) VALUES (
    '${AUTH_FIRST_PARTY_CLIENT_ID:-starter-app}',
    '',
    'Starter App (first-party)',
    ARRAY['${AUTH_ISSUER:-https://foundry.test}/auth/callback']::text[],
    ARRAY['openid','email','profile']::text[],
    true,
    true
)
ON CONFLICT (client_id) DO UPDATE
    SET redirect_uris = ARRAY['${AUTH_ISSUER:-https://foundry.test}/auth/callback']::text[],
        first_party   = true,
        public        = true,
        updated_at    = NOW();
