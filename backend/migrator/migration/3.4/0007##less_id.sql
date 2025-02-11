ALTER TABLE stage ADD COLUMN environment TEXT;
UPDATE stage SET environment = environment.resource_id FROM environment WHERE environment.id = stage.environment_id;
ALTER TABLE stage DROP COLUMN environment_id;
ALTER TABLE stage ALTER COLUMN environment SET NOT NULL;
ALTER TABLE stage ADD constraint stage_environment_fkey FOREIGN KEY (environment) references environment(resource_id);

ALTER TABLE project_webhook ADD COLUMN project TEXT;
UPDATE project_webhook SET project = project.resource_id FROM project WHERE project.id = project_webhook.project_id;
ALTER TABLE project_webhook DROP COLUMN project_id;
ALTER TABLE project_webhook ALTER COLUMN project SET NOT NULL;
ALTER TABLE project_webhook ADD constraint project_webhook_project_fkey FOREIGN KEY (project) references project(resource_id);

DROP INDEX IF EXISTS idx_db_instance_id;
DROP INDEX IF EXISTS idx_db_unique_instance_id_name;
DROP INDEX IF EXISTS idx_db_project_id;

ALTER TABLE db ADD COLUMN project TEXT;
UPDATE db SET project = project.resource_id FROM project WHERE project.id = db.project_id;
ALTER TABLE db DROP COLUMN project_id;
ALTER TABLE db ALTER COLUMN project SET NOT NULL;
ALTER TABLE db ADD constraint db_project_fkey FOREIGN KEY (project) references project(resource_id);

ALTER TABLE db ADD COLUMN instance TEXT;
UPDATE db SET instance = instance.resource_id FROM instance WHERE instance.id = db.instance_id;
ALTER TABLE db DROP COLUMN instance_id;
ALTER TABLE db ALTER COLUMN instance SET NOT NULL;
ALTER TABLE db ADD constraint db_instance_fkey FOREIGN KEY (instance) references instance(resource_id);

CREATE INDEX idx_db_project ON db(project);
CREATE UNIQUE INDEX idx_db_unique_instance_name ON db(instance, name);
