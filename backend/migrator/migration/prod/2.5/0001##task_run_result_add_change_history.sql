ALTER TABLE task_run DISABLE TRIGGER update_task_run_updated_ts;

UPDATE task_run
SET result = result || jsonb_build_object('changeHistory', concat('instances/', instance.resource_id, '/databases/', db.name, '/changeHistories/', instance_change_history.id))
FROM instance_change_history, instance, db
WHERE
    result->>'migrationId' IS NOT NULL
    AND (result->>'migrationId')::int = instance_change_history.id
    AND instance_change_history.instance_id = instance.id
    AND instance_change_history.database_id = db.id;

ALTER TABLE task_run ENABLE TRIGGER update_task_run_updated_ts;
