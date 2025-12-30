-- Migrate deprecated webhook activity types to new focused event types
-- Mappings:
--   ISSUE_CREATE → ISSUE_CREATED
--   ISSUE_APPROVAL_NOTIFY → ISSUE_APPROVAL_REQUESTED
--   ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE → PIPELINE_FAILED
--   ISSUE_PIPELINE_STAGE_STATUS_UPDATE → PIPELINE_COMPLETED
-- Removed (no mapping):
--   ISSUE_COMMENT_CREATE, ISSUE_FIELD_UPDATE, ISSUE_STATUS_UPDATE,
--   NOTIFY_ISSUE_APPROVED, NOTIFY_PIPELINE_ROLLOUT
UPDATE project_webhook
SET payload = jsonb_set(
  payload,
  '{activities}',
  (
    SELECT COALESCE(jsonb_agg(DISTINCT mapped_activity), '[]'::jsonb)
    FROM (
      SELECT
        CASE activity
          WHEN 'ISSUE_CREATE' THEN 'ISSUE_CREATED'
          WHEN 'ISSUE_APPROVAL_NOTIFY' THEN 'ISSUE_APPROVAL_REQUESTED'
          WHEN 'ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE' THEN 'PIPELINE_FAILED'
          WHEN 'ISSUE_PIPELINE_STAGE_STATUS_UPDATE' THEN 'PIPELINE_COMPLETED'
          ELSE activity
        END AS mapped_activity
      FROM jsonb_array_elements_text(payload->'activities') AS activity
      WHERE activity NOT IN (
        'ISSUE_COMMENT_CREATE',
        'ISSUE_FIELD_UPDATE',
        'ISSUE_STATUS_UPDATE',
        'NOTIFY_ISSUE_APPROVED',
        'NOTIFY_PIPELINE_ROLLOUT'
      )
    ) AS mapped
  )
)
WHERE payload ? 'activities';
