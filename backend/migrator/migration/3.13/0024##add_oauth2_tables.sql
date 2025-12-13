CREATE TABLE oauth2_client (
    client_id text PRIMARY KEY,
    client_secret_hash text NOT NULL,
    config jsonb NOT NULL,
    last_active_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE oauth2_authorization_code (
    code text PRIMARY KEY,
    client_id text NOT NULL REFERENCES oauth2_client(client_id) ON DELETE CASCADE,
    user_email text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    config jsonb NOT NULL,
    expires_at timestamptz NOT NULL
);

CREATE TABLE oauth2_refresh_token (
    token_hash text PRIMARY KEY,
    client_id text NOT NULL REFERENCES oauth2_client(client_id) ON DELETE CASCADE,
    user_email text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    expires_at timestamptz NOT NULL
);

CREATE INDEX idx_oauth2_authorization_code_expires_at ON oauth2_authorization_code(expires_at);
CREATE INDEX idx_oauth2_refresh_token_expires_at ON oauth2_refresh_token(expires_at);
CREATE INDEX idx_oauth2_client_last_active_at ON oauth2_client(last_active_at);
