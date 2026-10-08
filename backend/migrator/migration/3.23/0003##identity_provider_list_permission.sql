-- Listing identity providers now needs its own permission.
--
-- The list RPC used to be anonymous; it is now gated on
-- bb.identityProviders.list, which no existing role carries. The predefined
-- workspace admin role gains it in code, but custom roles live in this table:
-- any role that could already read a provider is the same role that reached
-- the SSO console, so it keeps working only if it also gets the new
-- permission. Idempotent, and no-ops on a fresh install.
UPDATE role
SET permissions = jsonb_set(
        permissions,
        '{permissions}',
        (permissions -> 'permissions') || '"bb.identityProviders.list"'::jsonb
    )
WHERE permissions -> 'permissions' @> '["bb.identityProviders.get"]'
  AND NOT permissions -> 'permissions' @> '["bb.identityProviders.list"]';
