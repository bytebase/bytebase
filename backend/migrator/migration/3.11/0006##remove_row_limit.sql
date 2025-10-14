-- Remove request.row_limit from IAM policy condition expressions
--
-- This migration removes the deprecated request.row_limit attribute from condition expressions
-- in IAM policies for SQL Editor User role bindings.
--
-- Background:
-- The request.row_limit condition was previously used to limit the number of rows that could be
-- queried by users with the SQL Editor User role. This functionality is being removed, and any
-- existing conditions that reference request.row_limit need to be cleaned up.
--
-- Changes:
-- - Remove " && request.row_limit <operator> <number>" patterns from condition expressions
-- - Remove "request.row_limit <operator> <number> && " patterns from condition expressions
-- - Remove standalone "request.row_limit <operator> <number>" when it's the only condition
-- - Supported operators: <, >, <=, >=, ==, !=
--
-- Example transformations:
-- Before: "request.time < timestamp(\"2025-10-18T06:05:51.000Z\") && request.row_limit <= 5000"
-- After:  "request.time < timestamp(\"2025-10-18T06:05:51.000Z\")"
--
-- Before: "request.row_limit <= 500"
-- After:  ""
--
-- Affected table:
-- - policy (type='IAM', resource_type='PROJECT'): IAM policy condition expressions for SQL Editor User role

UPDATE policy
SET payload = (
    SELECT jsonb_set(
        payload,
        '{bindings}',
        (
            SELECT jsonb_agg(
                CASE
                    -- Only update bindings for sqlEditorUser role that have conditions with request.row_limit
                    WHEN binding->>'role' = 'roles/sqlEditorUser'
                         AND binding->'condition'->>'expression' IS NOT NULL
                         AND binding->'condition'->>'expression' LIKE '%request.row_limit%'
                    THEN jsonb_set(
                        binding,
                        '{condition,expression}',
                        to_jsonb(
                            -- Remove standalone "request.row_limit <operator> <number>" pattern (when it's the only condition)
                            regexp_replace(
                                -- Remove " && request.row_limit <operator> <number>" pattern
                                regexp_replace(
                                    -- Remove "request.row_limit <operator> <number> && " pattern
                                    regexp_replace(
                                        binding->'condition'->>'expression',
                                        'request\.row_limit\s*(<=|>=|<|>|==|!=)\s*\d+\s*&&\s*',
                                        '',
                                        'g'
                                    ),
                                    '\s*&&\s*request\.row_limit\s*(<=|>=|<|>|==|!=)\s*\d+',
                                    '',
                                    'g'
                                ),
                                '^\s*request\.row_limit\s*(<=|>=|<|>|==|!=)\s*\d+\s*$',
                                '',
                                'g'
                            )
                        )
                    )
                    ELSE binding
                END
            )
            FROM jsonb_array_elements(payload->'bindings') AS binding
        )
    )
    FROM policy p2
    WHERE p2.id = policy.id
)
WHERE type = 'IAM'
  AND resource_type = 'PROJECT'
  AND payload->'bindings' IS NOT NULL
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements(payload->'bindings') AS binding
    WHERE binding->>'role' = 'roles/sqlEditorUser'
      AND binding->'condition'->>'expression' LIKE '%request.row_limit%'
  );
