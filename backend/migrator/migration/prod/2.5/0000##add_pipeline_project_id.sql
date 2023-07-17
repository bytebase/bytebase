ALTER TABLE pipeline ADD COLUMN project_id INTEGER REFERENCES project (id);

UPDATE pipeline
SET project_id = issue.project_id
FROM issue
WHERE issue.pipeline_id = pipeline.id;

DELETE FROM pipeline
WHERE pipeline.project_id IS NULL;

ALTER TABLE pipeline ALTER COLUMN project_id SET NOT NULL;