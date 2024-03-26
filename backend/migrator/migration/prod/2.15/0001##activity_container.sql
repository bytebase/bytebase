ALTER TABLE activity DISABLE TRIGGER update_activity_updated_ts;

ALTER TABLE activity ADD COLUMN resource_container TEXT;
CREATE INDEX idx_activity_resource_container ON activity(resource_container);

DELETE FROM activity WHERE type = 'bb.database.recovery.pitr.done';
UPDATE activity SET resource_container = 'projects/' || coalesce((SELECT resource_id FROM project WHERE project.id = activity.container_id), 'default') WHERE type LIKE '%project%';
UPDATE activity SET resource_container = 'projects/' || coalesce((SELECT project.resource_id FROM project JOIN issue ON project.id = issue.project_id WHERE issue.id = activity.container_id), 'default') WHERE type LIKE '%issue%';
UPDATE activity SET resource_container = 'projects/' || coalesce((SELECT project.resource_id FROM project JOIN pipeline ON project.id = pipeline.project_id WHERE pipeline.id = activity.container_id), 'default') WHERE type LIKE '%pipeline%';
UPDATE activity SET type = 'bb.sql.query' WHERE type = 'bb.sql-editor.query';
UPDATE activity SET resource_container = 'projects/' || coalesce((SELECT project.resource_id FROM project JOIN db ON project.id = db.project_id WHERE (db.id)::text = (activity.payload->'databaseId')::text), 'default') WHERE type LIKE '%sql%' AND (payload ? 'databaseId');
UPDATE activity SET resource_container = 'projects/default' WHERE type LIKE '%sql%' AND resource_container='';

ALTER TABLE activity ENABLE TRIGGER update_activity_updated_ts;