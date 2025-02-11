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
