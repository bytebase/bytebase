ALTER TABLE task_run ADD COLUMN sheet_id INTEGER;

ALTER TABLE task_run ADD CONSTRAINT task_run_sheet_id_fkey FOREIGN KEY (sheet_id) REFERENCES sheet (id);

UPDATE task_run
SET sheet_id = (
    SELECT
        (task.payload->>'sheetId')::INTEGER
    FROM task
    WHERE task.id = task_run.task_id
)