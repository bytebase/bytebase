-- Migrate IAM bindings based on environment DDL/DML policies.
-- No-op if all environments allow both DDL and DML.
-- Otherwise:
--   1. Replace sqlEditorUser with sqlEditorReadUser (keep existing conditions intact)
--   2. Add a new sqlEditorUser binding scoped to allowed environments
--   3. Scope custom roles with bb.sql.ddl/bb.sql.dml to allowed environments

WITH
-- Get all environment IDs from the setting table
all_envs AS (
    SELECT env->>'id' AS env_id
    FROM setting
    CROSS JOIN LATERAL jsonb_array_elements(COALESCE(value->'environments', '[]'::jsonb)) AS env
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
-- Custom roles with bb.sql.ddl or bb.sql.dml
custom_roles_with_env_limitation AS (
    SELECT 'roles/' || resource_id AS role_name
    FROM role
    WHERE permissions->'permissions' @> '["bb.sql.ddl"]'::jsonb
       OR permissions->'permissions' @> '["bb.sql.dml"]'::jsonb
)
UPDATE policy
SET payload = (
    SELECT jsonb_set(
        payload,
        '{bindings}',
        (
            SELECT COALESCE(jsonb_agg(new_binding ORDER BY ord), '[]'::jsonb)
            FROM (
                -- Existing bindings: swap sqlEditorUser -> sqlEditorReadUser,
                -- and add env condition to custom roles with DDL/DML
                SELECT rn AS ord,
                    CASE
                        -- sqlEditorUser: replace with sqlEditorReadUser
                        WHEN binding->>'role' = 'roles/sqlEditorUser'
                             AND COALESCE(binding->'condition'->>'expression', '') NOT LIKE '%resource.environment_id%'
                        THEN jsonb_set(binding, '{role}', '"roles/sqlEditorReadUser"')

                        -- Custom roles: add env condition
                        WHEN binding->>'role' IN (SELECT role_name FROM custom_roles_with_env_limitation)
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
                                ELSE jsonb_set(binding, '{condition,expression}', to_jsonb((binding->'condition'->>'expression') || ' && resource.environment_id in [' || (SELECT env_ids FROM env_list) || ']'))
                            END

                        ELSE binding
                    END AS new_binding
                FROM jsonb_array_elements(payload->'bindings') WITH ORDINALITY AS t(binding, rn)

                UNION ALL

                -- New sqlEditorUser bindings scoped to allowed environments
                SELECT 1000000 + rn AS ord,
                    CASE
                        WHEN binding->'condition' IS NULL OR COALESCE(binding->'condition'->>'expression', '') = ''
                        THEN jsonb_build_object(
                            'role', 'roles/sqlEditorUser',
                            'members', binding->'members',
                            'condition', jsonb_build_object('expression', 'resource.environment_id in [' || (SELECT env_ids FROM env_list) || ']')
                        )
                        ELSE jsonb_build_object(
                            'role', 'roles/sqlEditorUser',
                            'members', binding->'members',
                            'condition', jsonb_build_object('expression', (binding->'condition'->>'expression') || ' && resource.environment_id in [' || (SELECT env_ids FROM env_list) || ']')
                        )
                    END AS new_binding
                FROM jsonb_array_elements(payload->'bindings') WITH ORDINALITY AS t(binding, rn)
                WHERE binding->>'role' = 'roles/sqlEditorUser'
                  AND COALESCE(binding->'condition'->>'expression', '') NOT LIKE '%resource.environment_id%'
                  AND (SELECT env_ids FROM env_list) != ''
            ) sub
        )
    )
)
WHERE type = 'IAM'
  AND resource_type = 'PROJECT'
  AND payload->'bindings' IS NOT NULL
  -- Skip if all environments allow both DDL and DML
  AND EXISTS (
      SELECT 1 FROM all_envs e
      JOIN env_policy ep ON e.env_id = ep.env_id
      WHERE ep.disallow_ddl OR ep.disallow_dml
  );
