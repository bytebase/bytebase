ALTER TABLE task_run DISABLE TRIGGER update_task_run_updated_ts;

ALTER TABLE task_run ADD COLUMN IF NOT EXISTS attempt INTEGER;

DROP INDEX IF EXISTS uk_task_run_task_id_attempt;

UPDATE task_run
SET attempt = task_run_group.sequence - 1
FROM (
    SELECT id, row_number() over (PARTITION BY task_id ORDER BY id ASC) as sequence
    FROM task_run ORDER BY id ASC
) task_run_group
WHERE task_run.id = task_run_group.id;

ALTER TABLE task_run ALTER COLUMN attempt SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uk_task_run_task_id_attempt ON task_run (task_id, attempt);

ALTER TABLE task_run ENABLE TRIGGER update_task_run_updated_ts;
