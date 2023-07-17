ALTER TABLE pipeline ADD COLUMN project_id INTEGER REFERENCES project (id);

UPDATE pipeline
SET project_id = db.project_id
FROM task, db
WHERE task.type = 'bb.task.database.backup' AND task.pipeline_id = pipeline.id AND task.database_id = db.id;

UPDATE pipeline
SET project_id = issue.project_id
FROM issue
WHERE issue.pipeline_id = pipeline.id;

ALTER TABLE pipeline ALTER COLUMN project_id SET NOT NULL;
