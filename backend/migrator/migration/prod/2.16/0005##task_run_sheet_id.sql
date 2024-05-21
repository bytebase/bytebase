ALTER TABLE task_run ADD COLUMN sheet_id INTEGER;

ALTER TABLE task_run ADD CONSTRAINT task_run_sheet_id_fkey FOREIGN KEY (sheet_id) REFERENCES sheet (id);

WITH t AS (
    SELECT
        task_run.id AS task_run_id,
        sheet.id AS sheet_id
    FROM task_run
    LEFT JOIN task ON task.id = task_run.task_id
    LEFT JOIN sheet ON task.payload->>'sheetId' = (sheet.id)::TEXT
    WHERE sheet.id IS NOT NULL
)
UPDATE task_run
SET sheet_id = t.sheet_id
FROM t
WHERE task_run.id = t.task_run_id;