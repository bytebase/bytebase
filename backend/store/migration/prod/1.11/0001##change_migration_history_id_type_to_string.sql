UPDATE task_run
SET result = result || jsonb_build_object('migrationId', result->>'migrationId'::TEXT)
WHERE result ? 'migrationId';

