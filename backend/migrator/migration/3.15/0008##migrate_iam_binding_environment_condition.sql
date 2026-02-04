-- For roles with bb.sql.ddl or bb.sql.dml permissions (excluding projectOwner),
-- add "resource.environment_id in []" to the binding condition expression
-- if it doesn't already have one.

WITH roles_with_env_limitation AS (
    -- Predefined role with bb.sql.ddl/bb.sql.dml (excluding projectOwner)
    SELECT 'roles/sqlEditorUser' AS role_name
    UNION ALL
    -- Custom roles with bb.sql.ddl or bb.sql.dml
    SELECT 'roles/' || resource_id AS role_name
    FROM role
    WHERE resource_id != 'projectOwner'
    AND (
        permissions->'permissions' @> '["bb.sql.ddl"]'::jsonb
        OR permissions->'permissions' @> '["bb.sql.dml"]'::jsonb
    )
)
UPDATE policy
SET payload = (
    SELECT jsonb_set(
        payload,
        '{bindings}',
        (
            SELECT COALESCE(jsonb_agg(
                CASE
                    WHEN binding->>'role' IN (SELECT role_name FROM roles_with_env_limitation)
                         AND COALESCE(binding->'condition'->>'expression', '') NOT LIKE '%resource.environment_id%'
                    THEN
                        CASE
                            -- condition key doesn't exist: create it with expression
                            WHEN binding->'condition' IS NULL
                            THEN binding || jsonb_build_object('condition', jsonb_build_object('expression', 'resource.environment_id in []'))
                            -- condition exists but expression is empty or missing
                            WHEN COALESCE(binding->'condition'->>'expression', '') = ''
                            THEN jsonb_set(binding, '{condition,expression}', '"resource.environment_id in []"'::jsonb)
                            -- condition exists with non-empty expression: append
                            ELSE jsonb_set(
                                binding,
                                '{condition,expression}',
                                to_jsonb((binding->'condition'->>'expression') || ' && resource.environment_id in []')
                            )
                        END
                    ELSE binding
                END
            ), '[]'::jsonb)
            FROM jsonb_array_elements(payload->'bindings') AS binding
        )
    )
)
WHERE type = 'IAM'
  AND resource_type = 'PROJECT'
  AND payload->'bindings' IS NOT NULL;
