-- Add bb.databaseGroups.list and bb.databaseGroups.get to roles that have bb.projects.get
-- This maintains backward compatibility when migrating from bb.projects.get to bb.databaseGroups.list/get
UPDATE role
SET permissions = jsonb_set(
    permissions,
    '{permissions}',
    (
        SELECT jsonb_agg(DISTINCT p ORDER BY p)
        FROM (
            SELECT jsonb_array_elements_text(permissions->'permissions') AS p
            UNION
            SELECT 'bb.databaseGroups.get'
            UNION
            SELECT 'bb.databaseGroups.list'
        ) sub
    )
)
WHERE permissions->'permissions' ? 'bb.projects.get';

-- Add bb.databaseGroups.create, bb.databaseGroups.update, and bb.databaseGroups.delete to roles that have bb.projects.update
-- This maintains backward compatibility when migrating from bb.projects.update to bb.databaseGroups.create/update/delete
UPDATE role
SET permissions = jsonb_set(
    permissions,
    '{permissions}',
    (
        SELECT jsonb_agg(DISTINCT p ORDER BY p)
        FROM (
            SELECT jsonb_array_elements_text(permissions->'permissions') AS p
            UNION
            SELECT 'bb.databaseGroups.create'
            UNION
            SELECT 'bb.databaseGroups.delete'
            UNION
            SELECT 'bb.databaseGroups.update'
        ) sub
    )
)
WHERE permissions->'permissions' ? 'bb.projects.update';
