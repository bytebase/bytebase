CREATE TABLE web_refresh_token (
    token_hash  TEXT PRIMARY KEY,
    user_email  TEXT NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL,
    rotated_at  TIMESTAMPTZ
);

CREATE INDEX idx_web_refresh_token_user_email ON web_refresh_token(user_email);
CREATE INDEX idx_web_refresh_token_expires_at ON web_refresh_token(expires_at);
CREATE INDEX idx_web_refresh_token_rotated_at ON web_refresh_token(rotated_at) WHERE rotated_at IS NOT NULL;
