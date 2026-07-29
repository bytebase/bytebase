-- Persist the RFC 8707 resource indicator and the consented OAuth2 scope on
-- oauth2_refresh_token so a refresh carries the original grant forward
-- unchanged. The authorization code keeps both in its config JSONB
-- (OAuth2AuthorizationCodeConfig); the refresh token has flat columns.
--
-- Both are nullable: grants issued before this migration have neither, and a
-- client that omits the resource/scope parameters still consents successfully.

ALTER TABLE oauth2_refresh_token
    ADD COLUMN IF NOT EXISTS resource text,
    ADD COLUMN IF NOT EXISTS scope text;
