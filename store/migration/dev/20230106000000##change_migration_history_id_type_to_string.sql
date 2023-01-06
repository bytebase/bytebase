UPDATE task
SET payload = payload || jsonb_build_object('migrationID', payload->>'migrationID'::TEXT)
WHERE type = 'bb.task.database.data.update' AND payload ? 'migrationID';

UPDATE task_run
SET result = result || jsonb_build_object('migrationId', result->>'migrationId'::TEXT)
WHERE result ? 'migrationId';

