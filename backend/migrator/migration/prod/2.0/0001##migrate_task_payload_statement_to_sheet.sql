DO $$
DECLARE
  task_row RECORD;
  sheet_id INT;
BEGIN
FOR task_row IN (
  SELECT task.id, task.payload->>'statement' AS statement FROM task WHERE task.payload ? 'statement' ORDER BY task.id
)
LOOP

INSERT INTO
    sheet (creator_id, updater_id, project_id, database_id, name, statement, visibility, source, type, payload)
SELECT
    task.creator_id, task.creator_id, COALESCE(issue.project_id, 1), task.database_id, CONCAT('A sheet for issue #', issue.id), task_row.statement, 'PROJECT', 'BYTEBASE_ARTIFACT', 'SQL', '{}'
FROM task
LEFT JOIN issue ON task.pipeline_id = issue.pipeline_id
WHERE task.id = task_row.id
RETURNING sheet.id
INTO sheet_id;

UPDATE task
SET payload = task.payload - 'statement' || jsonb_build_object('sheetId', sheet_id)
WHERE task.id = task_row.id;

END LOOP;
END $$;

DO $$
DECLARE
  task_row RECORD;
  sheet_id INT;
BEGIN
FOR task_row IN (
  SELECT task.id, task.payload->>'rollbackStatement' AS statement FROM task WHERE task.payload ? 'rollbackStatement' ORDER BY task.id
)
LOOP

INSERT INTO
    sheet (creator_id, updater_id, project_id, database_id, name, statement, visibility, source, type, payload)
SELECT
    task.creator_id, task.creator_id, COALESCE(issue.project_id, 1), task.database_id, CONCAT('A rollback sheet for issue #', issue.id), task_row.statement, 'PROJECT', 'BYTEBASE_ARTIFACT', 'SQL', '{}'
FROM task
LEFT JOIN issue ON task.pipeline_id = issue.pipeline_id
WHERE task.id = task_row.id
RETURNING sheet.id
INTO sheet_id;

UPDATE task
SET payload = task.payload - 'rollbackStatement' || jsonb_build_object('rollbackSheetId', sheet_id)
WHERE task.id = task_row.id;

END LOOP;
END $$;
