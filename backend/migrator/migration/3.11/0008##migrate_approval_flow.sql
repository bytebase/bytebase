-- Migrate approval flow structure from nested ApprovalStep/ApprovalNode to flat roles array
--
-- This migration updates approval templates stored in the database to use the new flattened structure.
--
-- Old structure:
--   ApprovalFlow {
--     repeated ApprovalStep steps
--   }
--   ApprovalStep {
--     Type type
--     repeated ApprovalNode nodes
--   }
--   ApprovalNode {
--     Type type
--     string role
--   }
--
-- New structure:
--   ApprovalFlow {
--     repeated string roles
--   }
--
-- The migration extracts roles from the nested steps[].nodes[].role path and creates a flat roles[] array.
--
-- Affected data:
-- 1. setting table: WorkspaceApprovalSetting stored as text (JSON)
-- 2. issue table: Issue.payload JSONB containing approvalTemplates
--
-- Note: Since protojson.Marshal produces camelCased keys, we work with:
-- - "approvalTemplate" instead of "approval_template" (changed from array to singleton)
-- - "flow.steps" and "flow.nodes" in the old structure
-- - "flow.roles" in the new structure

-- Fix corrupted WORKSPACE_APPROVAL settings with null rules
-- This fixes a bug from 3.7/0001##approval_template_creator.sql where jsonb_agg()
-- returned NULL when processing empty rules arrays, resulting in {"rules": null}.
-- Remove the rules field entirely, leaving an empty object {}.
UPDATE setting
SET value = (value::jsonb - 'rules')::text
WHERE name = 'WORKSPACE_APPROVAL'
  AND jsonb_typeof(value::jsonb->'rules') = 'null';

-- Update workspace approval settings in the setting table
-- The setting.value is stored as text containing JSON
UPDATE setting
SET value = (
    SELECT jsonb_build_object(
        'rules',
        (
            SELECT jsonb_agg(
                jsonb_build_object(
                    'condition', rule->'condition',
                    'template', jsonb_build_object(
                        'title', rule->'template'->'title',
                        'description', rule->'template'->'description',
                        'flow', jsonb_build_object(
                            'roles',
                            COALESCE(
                                (
                                    -- Extract all roles from steps[].nodes[].role
                                    -- Only process if nodes is actually an array to avoid "cannot extract elements from a scalar" error
                                    SELECT jsonb_agg(node->>'role')
                                    FROM jsonb_array_elements(rule->'template'->'flow'->'steps') AS step,
                                         jsonb_array_elements(step->'nodes') AS node
                                    WHERE node->>'role' IS NOT NULL
                                      AND jsonb_typeof(step->'nodes') = 'array'
                                ),
                                '[]'::jsonb
                            )
                        )
                    )
                )
            )
            FROM jsonb_array_elements((value::jsonb)->'rules') AS rule
        )
    )::text
)
WHERE name = 'WORKSPACE_APPROVAL'
  AND value::jsonb->'rules' IS NOT NULL
  AND jsonb_typeof(value::jsonb->'rules') = 'array'
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements((value::jsonb)->'rules') AS rule
    WHERE rule->'template'->'flow'->'steps' IS NOT NULL
      AND jsonb_typeof(rule->'template'->'flow'->'steps') = 'array'
  );

-- Update issue approval template in the issue table
-- The issue.payload JSONB column contains approval.approvalTemplates (array)
-- Migrate to approval.approvalTemplate (singleton)
UPDATE issue
SET payload = jsonb_set(
    payload,
    '{approval}',
    (
        SELECT jsonb_set(
            (payload->'approval')::jsonb - 'approvalTemplates',
            '{approvalTemplate}',
            jsonb_strip_nulls(
                jsonb_build_object(
                    'title', (payload->'approval'->'approvalTemplates'->0)->'title',
                    'description', (payload->'approval'->'approvalTemplates'->0)->'description',
                    'flow', jsonb_build_object(
                        'roles',
                        COALESCE(
                            (
                                -- Extract all roles from steps[].nodes[].role
                                -- Only process if nodes is actually an array to avoid "cannot extract elements from a scalar" error
                                SELECT jsonb_agg(node->>'role')
                                FROM jsonb_array_elements((payload->'approval'->'approvalTemplates'->0)->'flow'->'steps') AS step,
                                     jsonb_array_elements(step->'nodes') AS node
                                WHERE node->>'role' IS NOT NULL
                                  AND jsonb_typeof(step->'nodes') = 'array'
                            ),
                            '[]'::jsonb
                        )
                    )
                )
            )
        )
    )
)
WHERE payload->'approval'->'approvalTemplates' IS NOT NULL
  AND jsonb_typeof(payload->'approval'->'approvalTemplates') = 'array'
  AND jsonb_array_length(payload->'approval'->'approvalTemplates') > 0
  AND (payload->'approval'->'approvalTemplates'->0)->'flow'->'steps' IS NOT NULL
  AND jsonb_typeof((payload->'approval'->'approvalTemplates'->0)->'flow'->'steps') = 'array';

-- Clean up: Remove orphaned approvalTemplates field from issues that weren't migrated
-- (e.g., empty arrays, malformed data)
UPDATE issue
SET payload = jsonb_set(
    payload,
    '{approval}',
    (payload->'approval')::jsonb - 'approvalTemplates'
)
WHERE payload->'approval' ? 'approvalTemplates'
  AND NOT payload->'approval' ? 'approvalTemplate'
  AND (
    jsonb_array_length(payload->'approval'->'approvalTemplates') = 0
    OR (payload->'approval'->'approvalTemplates'->0) IS NULL
    OR (payload->'approval'->'approvalTemplates'->0)->'flow' IS NULL
  );
