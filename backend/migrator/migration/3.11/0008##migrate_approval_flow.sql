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
                                    SELECT jsonb_agg(node->>'role')
                                    FROM jsonb_array_elements(rule->'template'->'flow'->'steps') AS step,
                                         jsonb_array_elements(step->'nodes') AS node
                                    WHERE node->>'role' IS NOT NULL
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
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements((value::jsonb)->'rules') AS rule
    WHERE rule->'template'->'flow'->'steps' IS NOT NULL
  );

-- Update issue approval template in the issue table
-- The issue.payload JSONB column contains approvalTemplate (singleton, changed from array)
-- First, migrate existing approvalTemplates[0] to approvalTemplate
UPDATE issue
SET payload = jsonb_set(
    payload - 'approvalTemplates',
    '{approvalTemplate}',
    (
        SELECT jsonb_strip_nulls(
            jsonb_build_object(
                'title', (payload->'approvalTemplates'->0)->'title',
                'description', (payload->'approvalTemplates'->0)->'description',
                'flow', jsonb_build_object(
                    'roles',
                    COALESCE(
                        (
                            -- Extract all roles from steps[].nodes[].role
                            SELECT jsonb_agg(node->>'role')
                            FROM jsonb_array_elements((payload->'approvalTemplates'->0)->'flow'->'steps') AS step,
                                 jsonb_array_elements(step->'nodes') AS node
                            WHERE node->>'role' IS NOT NULL
                        ),
                        '[]'::jsonb
                    )
                )
            )
        )
    )
)
WHERE payload->'approvalTemplates' IS NOT NULL
  AND jsonb_array_length(payload->'approvalTemplates') > 0
  AND (payload->'approvalTemplates'->0)->'flow'->'steps' IS NOT NULL;

-- Clean up: Remove orphaned approvalTemplates field from issues that weren't migrated
-- (e.g., empty arrays, malformed data)
UPDATE issue
SET payload = payload - 'approvalTemplates'
WHERE payload ? 'approvalTemplates'
  AND NOT payload ? 'approvalTemplate'
  AND (
    jsonb_array_length(payload->'approvalTemplates') = 0
    OR (payload->'approvalTemplates'->0) IS NULL
    OR (payload->'approvalTemplates'->0)->'flow' IS NULL
  );
