-- Split the admin backstop bb.savedQueries.manage into the per-verb
-- permissions it covered, minus re-share: get, getIamPolicy, update, delete.
-- Custom roles reach here from two populations: pre-release 3.22 builds that
-- created bb.savedQueries.manage directly, and every shipped release, whose
-- bb.worksheets.manage custom roles 0007 renames into it during the same
-- upgrade. setIamPolicy is deliberately not granted: sharing retires to the
-- creator for migrated custom roles exactly as it does for the predefined
-- admin roles, and granting it back is an explicit custom-role edit.
-- Predefined roles are code.
UPDATE role
SET permissions = jsonb_set(permissions, '{permissions}', (
    SELECT COALESCE(jsonb_agg(DISTINCT expanded.v), '[]'::jsonb)
    FROM jsonb_array_elements_text(permissions->'permissions') AS p,
        LATERAL unnest(CASE WHEN p = 'bb.savedQueries.manage'
            THEN ARRAY['bb.savedQueries.get', 'bb.savedQueries.getIamPolicy', 'bb.savedQueries.update', 'bb.savedQueries.delete']
            ELSE ARRAY[p]
        END) AS expanded(v)))
WHERE jsonb_typeof(permissions->'permissions') = 'array'
    AND permissions->'permissions' ? 'bb.savedQueries.manage';
