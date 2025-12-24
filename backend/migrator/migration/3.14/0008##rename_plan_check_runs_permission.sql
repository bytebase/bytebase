-- Rename bb.planCheckRuns.list to bb.planCheckRuns.get in custom roles
UPDATE role
SET permissions = jsonb_set(
    permissions,
    '{permissions}',
    (
        SELECT jsonb_agg(
            CASE
                WHEN elem = 'bb.planCheckRuns.list' THEN 'bb.planCheckRuns.get'
                ELSE elem
            END
        )
        FROM jsonb_array_elements_text(permissions->'permissions') AS elem
    )
)
WHERE permissions->'permissions' @> '"bb.planCheckRuns.list"';
