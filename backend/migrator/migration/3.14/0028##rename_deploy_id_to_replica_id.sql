-- Rename deployId to replicaId in task_run_log payload JSONB
UPDATE task_run_log
SET payload = payload - 'deployId' || jsonb_build_object('replicaId', payload->'deployId')
WHERE payload ? 'deployId';
