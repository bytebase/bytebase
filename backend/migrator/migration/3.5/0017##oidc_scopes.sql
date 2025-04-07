UPDATE idp
SET config = jsonb_set(
    config,
    '{scopes}',
    '["openid", "profile", "email"]'::jsonb,
    true
)
WHERE type = 'OIDC';
