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
-- JSONB keys are protojson camelCase, hence directorySyncToken / *Hash.

-- 1. Hash any real token that does not already have one. Reads the plaintext
--    before step 2 removes it.
UPDATE setting
SET value = jsonb_set(
        value,
        '{workspaceProfile,directorySyncTokenHash}',
        to_jsonb(
            encode(
                sha256(convert_to(value -> 'workspaceProfile' ->> 'directorySyncToken', 'UTF8')),
                'hex'
            )
        )
    )
WHERE name = 'WORKSPACE_PROFILE'
  AND coalesce(value -> 'workspaceProfile' ->> 'directorySyncToken', '') <> ''
  AND NOT (value -> 'workspaceProfile' ? 'directorySyncTokenHash');

-- 2. Drop the plaintext wherever it still appears, including the empty-string
--    rows step 1 skips and any row already carrying a hash.
UPDATE setting
SET value = value #- '{workspaceProfile,directorySyncToken}'
WHERE name = 'WORKSPACE_PROFILE'
  AND value -> 'workspaceProfile' ? 'directorySyncToken';
