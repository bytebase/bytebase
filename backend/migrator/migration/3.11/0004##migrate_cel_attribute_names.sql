-- Migrate CEL attribute names to use consistent resource.* prefix pattern
--
-- This migration updates all CEL expressions stored in the database to use the new naming convention
-- following Google AIP-140 standards for consistent snake_case field naming.
--
-- Attribute name changes (adding resource.* prefix):
-- 1. environment_id -> resource.environment_id
-- 2. project_id -> resource.project_id
-- 3. instance_id -> resource.instance_id
-- 4. db_engine -> resource.db_engine
-- 5. database_name -> resource.database_name
-- 6. schema_name -> resource.schema_name
-- 7. table_name -> resource.table_name
--
-- Attribute name changes (renaming for consistency):
-- 8. resource.environment_name -> resource.environment_id (in IAM policies)
-- 9. resource.schema -> resource.schema_name (in IAM policies)
-- 10. resource.table -> resource.table_name (in IAM policies)
-- 11. resource.labels -> resource.database_labels (in database groups)
--
-- Statement-related attribute changes (adding statement.* prefix):
-- 12. affected_rows -> statement.affected_rows
-- 13. table_rows -> statement.table_rows
-- 14. sql_type -> statement.sql_type
-- 15. sql_statement -> statement.text
--
-- Request-related attribute changes (adding request.* prefix):
-- 16. expiration_days -> request.expiration_days
-- 17. role -> request.role
--
-- Affected tables:
-- - risk: Risk evaluation CEL expressions
-- - db_group: Database group matching CEL expressions
-- - policy (type='IAM'): IAM policy condition expressions
-- - policy (type='MASKING_RULE'): Masking rule CEL expressions

-- Update risk table expressions
-- The risk.expression column stores CEL expressions in jsonb format with an "expression" field
-- We need to add resource.*, statement.*, and request.* prefixes to attributes that didn't have it
-- Using regexp_replace to match whole identifiers (not partial matches)
-- Note: We must do sql_statement -> statement.text BEFORE sql_type -> statement.sql_type to avoid partial match issues
UPDATE risk
SET expression = jsonb_set(
    expression,
    '{expression}',
    to_jsonb(
        regexp_replace(
            regexp_replace(
                regexp_replace(
                    regexp_replace(
                        regexp_replace(
                            regexp_replace(
                                regexp_replace(
                                    regexp_replace(
                                        regexp_replace(
                                            regexp_replace(
                                                regexp_replace(
                                                    regexp_replace(
                                                        regexp_replace(
                                                            expression->>'expression',
                                                            '\m(environment_id)\M',
                                                            'resource.environment_id',
                                                            'g'
                                                        ),
                                                        '\m(project_id)\M',
                                                        'resource.project_id',
                                                        'g'
                                                    ),
                                                    '\m(instance_id)\M',
                                                    'resource.instance_id',
                                                    'g'
                                                ),
                                                '\m(db_engine)\M',
                                                'resource.db_engine',
                                                'g'
                                            ),
                                            '\m(database_name)\M',
                                            'resource.database_name',
                                            'g'
                                        ),
                                        '\m(schema_name)\M',
                                        'resource.schema_name',
                                        'g'
                                    ),
                                    '\m(table_name)\M',
                                    'resource.table_name',
                                    'g'
                                ),
                                '\m(affected_rows)\M',
                                'statement.affected_rows',
                                'g'
                            ),
                            '\m(table_rows)\M',
                            'statement.table_rows',
                            'g'
                        ),
                        '\m(sql_statement)\M',
                        'statement.text',
                        'g'
                    ),
                    '\m(sql_type)\M',
                    'statement.sql_type',
                    'g'
                ),
                '\m(expiration_days)\M',
                'request.expiration_days',
                'g'
            ),
            '\mrole\M',
            'request.role',
            'g'
        )
    )
)
WHERE expression->>'expression' ~ '\m(environment_id|project_id|instance_id|db_engine|database_name|schema_name|table_name|affected_rows|table_rows|sql_statement|sql_type|expiration_days|role)\M'
  AND expression->>'expression' !~ 'resource\.(environment_id|project_id|instance_id|db_engine|database_name|schema_name|table_name)'
  AND expression->>'expression' !~ 'statement\.(affected_rows|table_rows|sql_type|text)'
  AND expression->>'expression' !~ 'request\.(expiration_days|role)';

-- Update database group expressions
-- The db_group.expression column stores CEL expressions in jsonb format with an "expression" field
UPDATE db_group
SET expression = jsonb_set(
    expression,
    '{expression}',
    to_jsonb(
        replace(
            expression->>'expression',
            'resource.labels',
            'resource.database_labels'
        )
    )
)
WHERE expression->>'expression' LIKE '%resource.labels%';

-- Update IAM policy conditions in policy table
-- The policy.payload column stores IAM policies with bindings that may have condition expressions
-- We need to update the expression field within condition objects
-- Changes: resource.environment_name -> resource.environment_id
--          resource.schema -> resource.schema_name (whole word match only, not resource.schema_name)
--          resource.table -> resource.table_name (whole word match only, not resource.table_name)
UPDATE policy
SET payload = jsonb_set(
    payload,
    '{bindings}',
    (
        SELECT jsonb_agg(
            CASE
                WHEN binding->'condition'->>'expression' IS NOT NULL
                     AND (binding->'condition'->>'expression' ~ '\mresource\.environment_name\M'
                          OR binding->'condition'->>'expression' ~ '\mresource\.schema\M'
                          OR binding->'condition'->>'expression' ~ '\mresource\.table\M')
                THEN jsonb_set(
                    binding,
                    '{condition,expression}',
                    to_jsonb(
                        regexp_replace(
                            regexp_replace(
                                regexp_replace(
                                    binding->'condition'->>'expression',
                                    '\mresource\.environment_name\M',
                                    'resource.environment_id',
                                    'g'
                                ),
                                '\mresource\.schema\M',
                                'resource.schema_name',
                                'g'
                            ),
                            '\mresource\.table\M',
                            'resource.table_name',
                            'g'
                        )
                    )
                )
                ELSE binding
            END
        )
        FROM jsonb_array_elements(policy.payload->'bindings') AS binding
    )
)
WHERE type = 'IAM'
  AND payload->'bindings' IS NOT NULL
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements(payload->'bindings') AS binding
    WHERE binding->'condition'->>'expression' ~ '\mresource\.environment_name\M'
       OR binding->'condition'->>'expression' ~ '\mresource\.schema\M'
       OR binding->'condition'->>'expression' ~ '\mresource\.table\M'
  );

-- Update masking rule policies in policy table
-- The policy.payload column for masking rules contains a rules array with condition expressions
-- Need to add resource.* prefix to attributes
UPDATE policy
SET payload = (
    SELECT jsonb_set(
        payload,
        '{rules}',
        (
            SELECT jsonb_agg(
                CASE
                    WHEN rule->'condition'->>'expression' IS NOT NULL
                    THEN jsonb_set(
                        rule,
                        '{condition,expression}',
                        to_jsonb(
                            regexp_replace(
                                regexp_replace(
                                    regexp_replace(
                                        regexp_replace(
                                            regexp_replace(
                                                regexp_replace(
                                                    regexp_replace(
                                                        regexp_replace(
                                                            rule->'condition'->>'expression',
                                                            '\m(environment_id)\M',
                                                            'resource.environment_id',
                                                            'g'
                                                        ),
                                                        '\m(project_id)\M',
                                                        'resource.project_id',
                                                        'g'
                                                    ),
                                                    '\m(instance_id)\M',
                                                    'resource.instance_id',
                                                    'g'
                                                ),
                                                '\m(database_name)\M',
                                                'resource.database_name',
                                                'g'
                                            ),
                                            '\m(schema_name)\M',
                                            'resource.schema_name',
                                            'g'
                                        ),
                                        '\m(table_name)\M',
                                        'resource.table_name',
                                        'g'
                                    ),
                                    '\m(column_name)\M',
                                    'resource.column_name',
                                    'g'
                                ),
                                '\m(classification_level)\M',
                                'resource.classification_level',
                                'g'
                            )
                        )
                    )
                    ELSE rule
                END
            )
            FROM jsonb_array_elements(payload->'rules') AS rule
        )
    )
    FROM policy p2
    WHERE p2.id = policy.id
)
WHERE type = 'MASKING_RULE'
  AND payload->'rules' IS NOT NULL
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements(payload->'rules') AS rule
    WHERE rule->'condition'->>'expression' ~ '\m(environment_id|project_id|instance_id|database_name|schema_name|table_name|column_name|classification_level)\M'
  );
