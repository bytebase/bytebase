-- Record when each session and grant row was issued.
--
-- The three token tables carry an expiry but no issue time, so nothing can
-- order a row against anything that happened to the account after it was
-- handed out. Existing rows default to migration time, which is the earliest
-- issue time consistent with their still being here.
ALTER TABLE web_refresh_token ADD COLUMN created_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE oauth2_authorization_code ADD COLUMN created_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE oauth2_refresh_token ADD COLUMN created_at timestamptz NOT NULL DEFAULT now();

-- Index the two foreign keys these tables carry. PostgreSQL indexes the
-- referenced side of a foreign key, never the referencing side, so both
-- cascades below are sequential scans over the whole table until they are.
--
-- user_email references principal(email) ON UPDATE CASCADE: every account
-- email change rewrites the rows keyed on it. web_refresh_token already has
-- this index; these two are catching up.
CREATE INDEX idx_oauth2_authorization_code_user_email ON oauth2_authorization_code(user_email);
CREATE INDEX idx_oauth2_refresh_token_user_email ON oauth2_refresh_token(user_email);

-- client_id references oauth2_client(client_id) ON DELETE CASCADE, and the
-- data cleaner deletes stale clients in bulk on a schedule. It answers no
-- query on its own — every lookup pairs it with the primary key.
CREATE INDEX idx_oauth2_authorization_code_client_id ON oauth2_authorization_code(client_id);
CREATE INDEX idx_oauth2_refresh_token_client_id ON oauth2_refresh_token(client_id);
