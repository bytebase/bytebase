-- Store the SCIM directory sync token as a SHA-256 hash instead of plaintext.
--
-- The token is a bearer credential that lived in plaintext inside
-- WORKSPACE_PROFILE, a setting every workspace member can read. The API no
-- longer returns it, and after this migration the plaintext is not retained
-- anywhere: only its hash is, so a leaked database yields nothing presentable.
--
-- The token VALUE is unchanged, so every configured Okta/Entra integration keeps
-- authenticating without the customer re-pasting anything.
--
-- SHA-256 rather than a slow KDF: the token is 122 bits from crypto/rand, not a
-- human-chosen secret, so key stretching buys nothing and would cost latency on
-- every SCIM request. sha256() is built into PostgreSQL 11+; no pgcrypto needed.
--
-- setting.value holds protojson of the setting message itself (UpsertSetting
-- marshals SettingMessage.Value directly), so WorkspaceProfileSetting's fields
-- are TOP-LEVEL camelCase keys. There is no "workspaceProfile" wrapper here —
-- that wrapper only exists in the v1 API, where the setting is carried inside
-- the SettingValue oneof.

-- 1. Hash any real token that does not already have one. Reads the plaintext
--    before step 2 removes it.
UPDATE setting
SET value = jsonb_set(
        value,
        '{directorySyncTokenHash}',
        to_jsonb(encode(sha256(convert_to(value ->> 'directorySyncToken', 'UTF8')), 'hex'))
    )
WHERE name = 'WORKSPACE_PROFILE'
  AND coalesce(value ->> 'directorySyncToken', '') <> ''
  AND NOT (value ? 'directorySyncTokenHash');

-- 2. Drop the plaintext wherever it still appears, including the empty-string
--    rows step 1 skips and any row already carrying a hash.
UPDATE setting
SET value = value - 'directorySyncToken'
WHERE name = 'WORKSPACE_PROFILE'
  AND value ? 'directorySyncToken';
