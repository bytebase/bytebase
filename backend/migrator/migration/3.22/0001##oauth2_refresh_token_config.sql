-- Give oauth2_refresh_token a config payload holding the consented grant state:
-- the RFC 8707 resource indicator and the OAuth2 scope, inherited from the
-- authorization code and carried forward unchanged by every refresh.
--
-- Stored as OAuth2RefreshTokenConfig (proto/store/store/oauth2.proto) rather than
-- flat columns: neither value uses a database feature (no FK, no NOT NULL, never
-- a query predicate) and both are read only from Go — the same shape as
-- oauth2_client.config and oauth2_authorization_code.config. It also lets the
-- grant gain fields later without another migration.
--
-- Defaults to '{}' so refresh tokens issued before this migration read back as an
-- empty grant, which the token endpoint treats as "no constraint to check".

ALTER TABLE oauth2_refresh_token
    ADD COLUMN IF NOT EXISTS config jsonb NOT NULL DEFAULT '{}';
