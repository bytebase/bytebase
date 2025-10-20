-- Drop the old check constraint that enforces "bb.plugin.webhook.*" format
ALTER TABLE project_webhook DROP CONSTRAINT IF EXISTS project_webhook_type_check;

-- Convert webhook type from "bb.plugin.webhook.*" format to enum names
UPDATE project_webhook SET type = 'SLACK' WHERE type = 'bb.plugin.webhook.slack';
UPDATE project_webhook SET type = 'DISCORD' WHERE type = 'bb.plugin.webhook.discord';
UPDATE project_webhook SET type = 'TEAMS' WHERE type = 'bb.plugin.webhook.teams';
UPDATE project_webhook SET type = 'DINGTALK' WHERE type = 'bb.plugin.webhook.dingtalk';
UPDATE project_webhook SET type = 'FEISHU' WHERE type = 'bb.plugin.webhook.feishu';
UPDATE project_webhook SET type = 'WECOM' WHERE type = 'bb.plugin.webhook.wecom';
UPDATE project_webhook SET type = 'LARK' WHERE type = 'bb.plugin.webhook.lark';

-- Convert event_list from "bb.webhook.event.*" format to enum names
UPDATE project_webhook
SET event_list = ARRAY(
    SELECT CASE event
        WHEN 'bb.webhook.event.issue.create' THEN 'ISSUE_CREATE'
        WHEN 'bb.webhook.event.issue.comment.create' THEN 'ISSUE_COMMENT_CREATE'
        WHEN 'bb.webhook.event.issue.update' THEN 'ISSUE_FIELD_UPDATE'
        WHEN 'bb.webhook.event.issue.status.update' THEN 'ISSUE_STATUS_UPDATE'
        WHEN 'bb.webhook.event.stage.status.update' THEN 'ISSUE_PIPELINE_STAGE_STATUS_UPDATE'
        WHEN 'bb.webhook.event.issue.approval.create' THEN 'ISSUE_APPROVAL_NOTIFY'
        WHEN 'bb.webhook.event.taskRun.status.update' THEN 'ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE'
        WHEN 'bb.webhook.event.issue.approval.pass' THEN 'NOTIFY_ISSUE_APPROVED'
        WHEN 'bb.webhook.event.issue.rollout.ready' THEN 'NOTIFY_PIPELINE_ROLLOUT'
        ELSE event  -- Keep unknown events as-is
    END
    FROM unnest(event_list) AS event
)
WHERE EXISTS (
    SELECT 1 FROM unnest(event_list) AS event
    WHERE event LIKE 'bb.webhook.event.%'
);
