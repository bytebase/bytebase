-- Migrate project_webhook.activity_list from base.ActivityXXX to base.EventTypeXXX
UPDATE project_webhook
SET activity_list = ARRAY(
    SELECT CASE
        WHEN unnest = 'bb.issue.create' THEN 'bb.webhook.event.issue.create'
        WHEN unnest = 'bb.issue.field.update' THEN 'bb.webhook.event.issue.update'
        WHEN unnest = 'bb.issue.status.update' THEN 'bb.webhook.event.issue.status.update'
        WHEN unnest = 'bb.issue.comment.create' THEN 'bb.webhook.event.issue.comment.create'
        WHEN unnest = 'bb.issue.approval.notify' THEN 'bb.webhook.event.issue.approval.create'
        WHEN unnest = 'bb.notify.issue.approved' THEN 'bb.webhook.event.issue.approval.pass'
        WHEN unnest = 'bb.notify.pipeline.rollout' THEN 'bb.webhook.event.issue.rollout.ready'
        WHEN unnest = 'bb.pipeline.stage.status.update' THEN 'bb.webhook.event.stage.status.update'
        WHEN unnest = 'bb.pipeline.taskrun.status.update' THEN 'bb.webhook.event.taskRun.status.update'
        ELSE unnest
    END
    FROM unnest(activity_list)
)
WHERE activity_list && ARRAY[
    'bb.issue.create',
    'bb.issue.field.update', 
    'bb.issue.status.update',
    'bb.issue.comment.create',
    'bb.issue.approval.notify',
    'bb.notify.issue.approved',
    'bb.notify.pipeline.rollout',
    'bb.pipeline.stage.status.update',
    'bb.pipeline.taskrun.status.update'
];