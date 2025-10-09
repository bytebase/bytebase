-- Cleanup WORKSPACE level IAM policies
--
-- This migration performs two cleanup operations on WORKSPACE level IAM policies:
-- 1. Remove projectExporter role and merge row_limit condition into sqlEditorUser role
-- 2. Remove request.row_limit from IAM policy condition expressions
--
-- Background:
-- The projectExporter role was removed at PROJECT level in migration 3.10/0007, but WORKSPACE
-- level policies were not handled at that time. Similarly, request.row_limit conditions were
-- cleaned up at PROJECT level in migration 3.11/0006, but WORKSPACE level was not handled.

-- Step 1: Remove projectExporter role and merge row_limit condition into sqlEditorUser role
WITH exporter_data AS (
    -- Collect all projectExporter members and their row_limit conditions
    SELECT
        p.id,
        m.member,
        substring(b.binding->'condition'->>'expression' from 'request\.row_limit\s*<=\s*\d+') as row_limit
    FROM policy p,
        jsonb_array_elements(p.payload->'bindings') b(binding),
        jsonb_array_elements_text(b.binding->'members') m(member)
    WHERE p.resource_type = 'WORKSPACE'
        AND p.type = 'IAM'
        AND b.binding->>'role' = 'roles/projectExporter'
        AND b.binding->'condition'->>'expression' ~ 'request\.row_limit\s*<=\s*\d+'
)
UPDATE policy p
SET payload = jsonb_set(
    p.payload,
    '{bindings}',
    (
        SELECT jsonb_agg(
            CASE
                -- Update sqlEditorUser bindings with row_limit if member exists in exporter_data
                WHEN b.binding->>'role' = 'roles/sqlEditorUser' AND EXISTS (
                    SELECT 1 FROM exporter_data ed
                    WHERE ed.id = p.id
                        AND ed.member IN (SELECT jsonb_array_elements_text(b.binding->'members'))
                ) THEN
                    jsonb_set(
                        b.binding,
                        '{condition,expression}',
                        to_jsonb(
                            COALESCE(b.binding->'condition'->>'expression', '') ||
                            CASE
                                WHEN b.binding->'condition'->>'expression' IS NOT NULL THEN ' && '
                                ELSE ''
                            END ||
                            (SELECT ed.row_limit FROM exporter_data ed
                             WHERE ed.id = p.id
                                AND ed.member IN (SELECT jsonb_array_elements_text(b.binding->'members'))
                             LIMIT 1)
                        )
                    )
                -- Keep all other non-projectExporter bindings unchanged
                ELSE b.binding
            END
        )
        FROM jsonb_array_elements(p.payload->'bindings') b(binding)
        WHERE b.binding->>'role' != 'roles/projectExporter'
    )
)
WHERE p.resource_type = 'WORKSPACE'
    AND p.type = 'IAM'
    AND EXISTS (
        SELECT 1
        FROM jsonb_array_elements(p.payload->'bindings') b(binding)
        WHERE b.binding->>'role' = 'roles/projectExporter'
    );

-- Step 2: Remove request.row_limit from IAM policy condition expressions
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
  AND resource_type = 'WORKSPACE'
  AND payload->'bindings' IS NOT NULL
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements(payload->'bindings') AS binding
    WHERE binding->>'role' = 'roles/sqlEditorUser'
      AND binding->'condition'->>'expression' LIKE '%request.row_limit%'
  );
