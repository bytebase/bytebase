ALTER TABLE plan ADD COLUMN last_plan_editor text;

-- Preserve existing approval behavior for every existing project, including
-- soft-deleted projects. New projects retain the proto default (false).
UPDATE project
SET setting = jsonb_set(setting, '{allowLastPlanEditorApproval}', 'true'::jsonb, true);
