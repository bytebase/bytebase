ALTER TABLE vcs DROP COLUMN application_id;
ALTER TABLE vcs DROP COLUMN secret;
ALTER TABLE vcs DROP COLUMN api_url;
ALTER TABLE vcs ADD COLUMN access_token TEXT NOT NULL DEFAULT '';
ALTER TABLE repository DROP COLUMN sheet_path_template;
ALTER TABLE repository DROP COLUMN access_token;
ALTER TABLE repository DROP COLUMN expires_ts;
ALTER TABLE repository DROP COLUMN refresh_token;
