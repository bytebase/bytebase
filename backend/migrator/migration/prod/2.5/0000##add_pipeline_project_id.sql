ALTER TABLE pipeline ADD COLUMN project_id INTEGER REFERENCES project (id);

UPDATE pipeline
SET project_id = db.project_id
FROM task, db
WHERE task.type = 'bb.task.database.backup' AND task.pipeline_id = pipeline.id AND task.database_id = db.id;

UPDATE pipeline
SET project_id = issue.project_id
FROM issue
WHERE issue.pipeline_id = pipeline.id;

DO $$
BEGIN
    IF (SELECT COUNT(*) FROM pipeline WHERE project_id IS NULL) < 10 THEN
        DELETE FROM task WHERE task.pipeline_id IN (SELECT pipeline.id FROM pipeline WHERE pipeline.project_id IS NULL);

        DELETE FROM stage WHERE stage.pipeline_id IN (SELECT pipeline.id FROM pipeline WHERE pipeline.project_id IS NULL);

        DELETE FROM pipeline WHERE project_id IS NULL;
    END IF;
END$$;

ALTER TABLE pipeline ALTER COLUMN project_id SET NOT NULL;
