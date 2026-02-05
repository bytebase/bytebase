-- For roles with bb.sql.ddl or bb.sql.dml permissions (excluding projectOwner),
-- add "resource.environment_id in [...]" to the binding condition expression
-- if it doesn't already have one.
-- The environment list is populated from existing QUERY_DATA policies:
-- environments where DDL or DML is not disallowed are included.

WITH
-- Get all environment IDs from the setting table
all_envs AS (
    SELECT env->>'id' AS env_id
    FROM setting, jsonb_array_elements(value->'environments') AS env
    WHERE name = 'ENVIRONMENT'
),
-- Get environment QUERY_DATA policies with DDL/DML flags
env_policy AS (
    SELECT
        SUBSTRING(resource FROM 'environments/(.+)') AS env_id,
        COALESCE((payload->>'disallowDdl')::boolean, false) AS disallow_ddl,
        COALESCE((payload->>'disallowDml')::boolean, false) AS disallow_dml
    FROM policy
    WHERE resource_type = 'ENVIRONMENT'
      AND type = 'QUERY_DATA'
),
-- Environments that allow DDL or DML (or don't have a QUERY_DATA policy)
allowed_envs AS (
    SELECT e.env_id
    FROM all_envs e
    LEFT JOIN env_policy ep ON e.env_id = ep.env_id
    WHERE ep.env_id IS NULL       -- No policy, default to allowed
       OR NOT ep.disallow_ddl     -- DDL is allowed
       OR NOT ep.disallow_dml     -- DML is allowed
),
-- Build the environment list as a CEL-compatible string like "env1", "env2"
env_list AS (
    SELECT COALESCE(string_agg('"' || env_id || '"', ', '), '') AS env_ids
    FROM allowed_envs
),
roles_with_env_limitation AS (
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
                            THEN binding || jsonb_build_object('condition', jsonb_build_object('expression', 'resource.environment_id in [' || (SELECT env_ids FROM env_list) || ']'))
                            -- condition exists but expression is empty or missing
                            WHEN COALESCE(binding->'condition'->>'expression', '') = ''
                            THEN jsonb_set(binding, '{condition,expression}', to_jsonb('resource.environment_id in [' || (SELECT env_ids FROM env_list) || ']'))
                            -- condition exists with non-empty expression: append
                            ELSE jsonb_set(
                                binding,
                                '{condition,expression}',
                                to_jsonb((binding->'condition'->>'expression') || ' && resource.environment_id in [' || (SELECT env_ids FROM env_list) || ']')
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
