ALTER TABLE activity ADD COLUMN resource_container TEXT;
CREATE INDEX idx_activity_resource_container ON activity(resource_container);

DELETE FROM activity WHERE type = 'bb.database.recovery.pitr.done';
UPDATE activity SET resource_container = 'projects/' || coalesce((SELECT resource_id FROM project WHERE project.id = activity.container_id), 'default') WHERE type LIKE '%project%';
UPDATE activity SET resource_container = 'projects/' || coalesce((SELECT project.resource_id FROM project JOIN issue ON project.id = issue.project_id WHERE issue.id = activity.container_id), 'default') WHERE type LIKE '%issue%';
UPDATE activity SET resource_container = 'projects/' || coalesce((SELECT project.resource_id FROM project JOIN pipeline ON project.id = pipeline.project_id WHERE pipeline.id = activity.container_id), 'default') WHERE type LIKE '%pipeline%';
