-- Move workspace binding from oauth2_client to oauth2_authorization_code and
-- oauth2_refresh_token. The workspace is now selected by the user at consent
-- time (Pattern A: token = workspace), so clients can be registered without
-- prior workspace context — enabling unauthenticated Dynamic Client
-- Registration on SaaS (RFC 7591). The discovery endpoint can then return
-- workspace-agnostic /api/oauth2/* URLs that work for any caller.

ALTER TABLE oauth2_client ALTER COLUMN workspace DROP NOT NULL;

ALTER TABLE oauth2_authorization_code
    ADD COLUMN IF NOT EXISTS workspace text REFERENCES workspace(resource_id);

ALTER TABLE oauth2_refresh_token
    ADD COLUMN IF NOT EXISTS workspace text REFERENCES workspace(resource_id);
