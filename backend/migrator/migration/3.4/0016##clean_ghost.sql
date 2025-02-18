WITH t1 AS (
    SELECT
        to_task.id AS cutover_task_id,
        from_task.payload AS sync_task_payload
    FROM task_dag
    LEFT JOIN task from_task ON from_task.id = task_dag.from_task_id
    LEFT JOIN task to_task ON to_task.id = task_dag.to_task_id
    WHERE to_task.type = 'bb.task.database.schema.update.ghost.cutover'
)
UPDATE task
SET payload = t1.sync_task_payload
FROM t1
WHERE id = t1.cutover_task_id;

DELETE FROM task_run_log
USING task_run, task
WHERE task_run_log.task_run_id = task_run.id AND task_run.task_id = task.id AND task.type = 'bb.task.database.schema.update.ghost.sync';

DELETE FROM task_run
USING task 
WHERE task.id = task_run.task_id AND task.type = 'bb.task.database.schema.update.ghost.sync';

DELETE FROM task_dag
USING task
WHERE task_dag.from_task_id = task.id AND task.type = 'bb.task.database.schema.update.ghost.sync';

DELETE FROM task
WHERE task.type = 'bb.task.database.schema.update.ghost.sync';

UPDATE task
SET type = 'bb.task.database.schema.update-ghost'
WHERE task.type = 'bb.task.database.schema.update.ghost.cutover';

UPDATE task_run
SET status = 'CANCELED'
FROM task
WHERE task.id = task_run.task_id AND task.type = 'bb.task.database.schema.update-ghost' AND task_run.status IN ('PENDING', 'RUNNING');
