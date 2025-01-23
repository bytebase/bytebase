ALTER TABLE project_webhook DISABLE TRIGGER update_project_webhook_updated_ts;

UPDATE project_webhook
SET activity_list = array_remove(activity_list, 'bb.pipeline.task.status.update')
WHERE activity_list @> '{bb.pipeline.task.status.update}';

ALTER TABLE project_webhook ENABLE TRIGGER update_project_webhook_updated_ts;
