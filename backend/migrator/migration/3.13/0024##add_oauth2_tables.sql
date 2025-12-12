CREATE TABLE oauth2_client (
    client_id TEXT PRIMARY KEY,
    client_secret_hash TEXT NOT NULL,
    config JSONB NOT NULL,
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE oauth2_authorization_code (
    code TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES oauth2_client(client_id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES principal(id),
    config JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE oauth2_refresh_token (
    token_hash TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES oauth2_client(client_id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES principal(id),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_oauth2_authorization_code_expires_at ON oauth2_authorization_code(expires_at);
CREATE INDEX idx_oauth2_refresh_token_expires_at ON oauth2_refresh_token(expires_at);
CREATE INDEX idx_oauth2_client_last_active_at ON oauth2_client(last_active_at);
