-- Remove bb.sql.export permission from all roles
UPDATE role
SET permissions = jsonb_set(
    permissions,
    '{permissions}',
    COALESCE((SELECT jsonb_agg(p) FROM jsonb_array_elements(permissions->'permissions') p WHERE p::text != '"bb.sql.export"'), '[]'::jsonb)
)
WHERE permissions->'permissions' ? 'bb.sql.export';