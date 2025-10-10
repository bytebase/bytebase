-- Add id field to ApprovalTemplate and identify built-in vs custom flows
--
-- This migration adds the 'id' field to all approval templates and assigns appropriate IDs:
-- - Built-in flows get "bb.*" prefix (e.g., "bb.project-owner", "bb.workspace-dba")
-- - Custom flows get a generated UUID
--
-- Detection logic for built-in flows:
-- Built-in flows are identified by their role combinations:
-- - bb.project-owner: [roles/projectOwner]
-- - bb.workspace-dba: [roles/workspaceDBA]
-- - bb.workspace-admin: [roles/workspaceAdmin]
-- - bb.project-owner-workspace-dba: [roles/projectOwner, roles/workspaceDBA]
-- - bb.project-owner-workspace-dba-workspace-admin: [roles/projectOwner, roles/workspaceDBA, roles/workspaceAdmin]
--
-- This migration also cleans up, trims, and merges approval templates:
-- - Step 1: Removes unused templates (empty/null conditions) upfront
-- - Step 2: Adds IDs to remaining templates
-- - Step 3: Deduplicates templates with the same ID
-- - Step 4: Merges conditions for duplicate templates using OR logic
--
-- Affected data:
-- 1. setting table: WorkspaceApprovalSetting stored as text (JSON)
-- 2. issue table: Issue.payload JSONB containing approvalTemplate
--
-- Note: Since protojson.Marshal produces camelCased keys, we work with:
-- - "approvalTemplate" and "flow.roles"

-- Helper function to detect built-in flow ID based on roles array
CREATE OR REPLACE FUNCTION detect_builtin_flow_id(roles jsonb)
RETURNS text AS $$
DECLARE
    roles_array text[];
    sorted_roles text;
BEGIN
    -- Extract roles as plain text (without quotes) and sort them
    SELECT array_agg(value ORDER BY value)
    INTO roles_array
    FROM jsonb_array_elements_text(roles);

    sorted_roles := array_to_string(roles_array, ',');

    -- Match against known built-in flow patterns
    CASE sorted_roles
        WHEN 'roles/projectOwner' THEN
            RETURN 'bb.project-owner';
        WHEN 'roles/workspaceDBA' THEN
            RETURN 'bb.workspace-dba';
        WHEN 'roles/workspaceAdmin' THEN
            RETURN 'bb.workspace-admin';
        WHEN 'roles/projectOwner,roles/workspaceDBA' THEN
            RETURN 'bb.project-owner-workspace-dba';
        WHEN 'roles/projectOwner,roles/workspaceAdmin,roles/workspaceDBA' THEN
            RETURN 'bb.project-owner-workspace-dba-workspace-admin';
        ELSE
            -- Not a built-in flow, return NULL to indicate custom flow
            RETURN NULL;
    END CASE;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Update workspace approval settings in the setting table
UPDATE setting
SET value = (
    WITH
    -- Step 1: Filter out rules with empty or null conditions (unused templates)
    rules_with_valid_conditions AS (
        SELECT rule
        FROM jsonb_array_elements((value::jsonb)->'rules') AS rule
        WHERE rule->'condition' IS NOT NULL
          AND rule->'condition'::text != 'null'
          AND rule->'condition'->>'expression' IS NOT NULL
          AND rule->'condition'->>'expression' != ''
    ),
    -- Step 2: Add IDs to all remaining templates
    rules_with_ids AS (
        SELECT
            rule->'condition' AS condition,
            jsonb_build_object(
                'id', COALESCE(
                    detect_builtin_flow_id(rule->'template'->'flow'->'roles'),
                    COALESCE(rule->'template'->>'id', gen_random_uuid()::text)
                ),
                'flow', rule->'template'->'flow',
                'title', rule->'template'->'title',
                'description', rule->'template'->'description'
            ) AS template
        FROM rules_with_valid_conditions
    ),
    -- Step 3: Find unique templates by deduplicating on template ID
    unique_templates AS (
        SELECT DISTINCT ON (template->>'id')
            template->>'id' AS template_id,
            template
        FROM rules_with_ids
    ),
    -- Step 4: Group conditions by template_id and merge expressions with OR logic
    merged_conditions AS (
        SELECT
            template->>'id' AS template_id,
            -- Merge multiple conditions for the same template using OR logic
            -- If there's only one condition, use it as-is
            -- If there are multiple, join with || without extra parentheses
            CASE
                WHEN COUNT(*) = 1 THEN
                    MAX(condition->>'expression')
                ELSE
                    string_agg(condition->>'expression', ' || ' ORDER BY condition->>'expression')
            END AS merged_expression
        FROM rules_with_ids
        GROUP BY template->>'id'
    ),
    -- Step 5: Build final rules array with merged conditions
    final_rules AS (
        SELECT jsonb_build_object(
            'condition', jsonb_build_object(
                'expression', m.merged_expression
            ),
            'template', t.template
        ) AS rule
        FROM merged_conditions m
        JOIN unique_templates t ON m.template_id = t.template_id
    )
    SELECT jsonb_build_object(
        'rules',
        COALESCE((SELECT jsonb_agg(rule) FROM final_rules), '[]'::jsonb)
    )::text
)
WHERE name = 'WORKSPACE_APPROVAL'
  AND value::jsonb->'rules' IS NOT NULL;

-- Update issue approval template in the issue table
UPDATE issue
SET payload = jsonb_set(
    payload,
    '{approval,approvalTemplate}',
    jsonb_build_object(
        'id', COALESCE(
            detect_builtin_flow_id((payload->'approval'->'approvalTemplate'->'flow'->'roles')::jsonb),
            -- If not a built-in flow, preserve existing ID or generate new UUID
            COALESCE(payload->'approval'->'approvalTemplate'->>'id', gen_random_uuid()::text)
        ),
        'flow', payload->'approval'->'approvalTemplate'->'flow',
        'title', payload->'approval'->'approvalTemplate'->'title',
        'description', payload->'approval'->'approvalTemplate'->'description'
    )
)
WHERE payload->'approval'->'approvalTemplate' IS NOT NULL
  AND payload->'approval'->'approvalTemplate'->'flow'->'roles' IS NOT NULL;

-- Clean up: Drop the helper function
DROP FUNCTION IF EXISTS detect_builtin_flow_id(jsonb);
