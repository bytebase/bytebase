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
CREATE INDEX idx_project_webhook_project ON project_webhook(project);

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

DROP INDEX IF EXISTS idx_db_schema_unique_database_id;
ALTER TABLE db_schema ADD COLUMN instance TEXT;
ALTER TABLE db_schema ADD COLUMN db_name TEXT;
UPDATE db_schema SET instance = db.instance, db_name = db.name FROM db WHERE db.id = db_schema.database_id;
ALTER TABLE db_schema ALTER COLUMN instance SET NOT NULL;
ALTER TABLE db_schema ALTER COLUMN db_name SET NOT NULL;
ALTER TABLE db_schema ADD constraint db_schema_instance_db_name_fkey FOREIGN KEY (instance, db_name) references db(instance, name);
ALTER TABLE db_schema DROP COLUMN database_id;
CREATE UNIQUE INDEX idx_db_schema_unique_instance_db_name ON db_schema(instance, db_name);

DROP INDEX IF EXISTS idx_data_source_unique_instance_id_name;
ALTER TABLE data_source ADD COLUMN instance TEXT;
UPDATE data_source SET instance = instance.resource_id FROM instance WHERE instance.id = data_source.instance_id;
ALTER TABLE data_source DROP COLUMN instance_id;
ALTER TABLE data_source ALTER COLUMN instance SET NOT NULL;
ALTER TABLE data_source ADD constraint data_source_instance_fkey FOREIGN KEY (instance) references instance(resource_id);
CREATE UNIQUE INDEX idx_data_source_unique_instance_name ON data_source(instance, name);

ALTER TABLE sheet ADD COLUMN project TEXT;
UPDATE sheet SET project = project.resource_id FROM project WHERE project.id = sheet.project_id;
ALTER TABLE sheet DROP COLUMN project_id;
ALTER TABLE sheet ALTER COLUMN project SET NOT NULL;
ALTER TABLE sheet ADD constraint sheet_project_fkey FOREIGN KEY (project) references project(resource_id);
CREATE INDEX idx_sheet_project ON sheet(project);

ALTER TABLE pipeline ADD COLUMN project TEXT;
UPDATE pipeline SET project = project.resource_id FROM project WHERE project.id = pipeline.project_id;
ALTER TABLE pipeline DROP COLUMN project_id;
ALTER TABLE pipeline ALTER COLUMN project SET NOT NULL;
ALTER TABLE pipeline ADD constraint pipeline_project_fkey FOREIGN KEY (project) references project(resource_id);

ALTER TABLE task ADD COLUMN instance TEXT;
UPDATE task SET instance = instance.resource_id FROM instance WHERE instance.id = task.instance_id;
ALTER TABLE task DROP COLUMN instance_id;
ALTER TABLE task ALTER COLUMN instance SET NOT NULL;
ALTER TABLE task ADD constraint task_instance_fkey FOREIGN KEY (instance) references instance(resource_id);

ALTER TABLE task ADD COLUMN db_name TEXT;
UPDATE task SET db_name = db.name FROM db WHERE db.id = task.database_id;
ALTER TABLE task DROP COLUMN database_id;

ALTER TABLE plan ADD COLUMN project TEXT;
UPDATE plan SET project = project.resource_id FROM project WHERE project.id = plan.project_id;
ALTER TABLE plan DROP COLUMN project_id;
ALTER TABLE plan ALTER COLUMN project SET NOT NULL;
ALTER TABLE plan ADD constraint plan_project_fkey FOREIGN KEY (project) references project(resource_id);
CREATE INDEX idx_plan_project ON plan(project);

ALTER TABLE issue ADD COLUMN project TEXT;
UPDATE issue SET project = project.resource_id FROM project WHERE project.id = issue.project_id;
ALTER TABLE issue DROP COLUMN project_id;
ALTER TABLE issue ALTER COLUMN project SET NOT NULL;
ALTER TABLE issue ADD constraint issue_project_fkey FOREIGN KEY (project) references project(resource_id);
CREATE INDEX idx_issue_project ON issue(project);

ALTER TABLE vcs_connector ADD COLUMN vcs TEXT;
UPDATE vcs_connector SET vcs = vcs.resource_id FROM vcs WHERE vcs.id = vcs_connector.vcs_id;
ALTER TABLE vcs_connector DROP COLUMN vcs_id;
ALTER TABLE vcs_connector ALTER COLUMN vcs SET NOT NULL;
ALTER TABLE vcs_connector ADD constraint vcs_connector_vcs_fkey FOREIGN KEY (project) references project(resource_id);

ALTER TABLE vcs_connector ADD COLUMN project TEXT;
UPDATE vcs_connector SET project = project.resource_id FROM project WHERE project.id = vcs_connector.project_id;
ALTER TABLE vcs_connector DROP COLUMN project_id;
ALTER TABLE vcs_connector ALTER COLUMN project SET NOT NULL;
ALTER TABLE vcs_connector ADD constraint vcs_connector_project_fkey FOREIGN KEY (project) references project(resource_id);

DELETE FROM anomaly WHERE database_id IS NULL;
DROP INDEX IF EXISTS idx_anomaly_unique_project_database_id_type;
ALTER TABLE anomaly ADD COLUMN instance TEXT;
ALTER TABLE anomaly ADD COLUMN db_name TEXT;
UPDATE anomaly SET instance = db.instance, db_name = db.name FROM db WHERE db.id = anomaly.database_id;
ALTER TABLE anomaly ALTER COLUMN instance SET NOT NULL;
ALTER TABLE anomaly ALTER COLUMN db_name SET NOT NULL;
ALTER TABLE anomaly ADD constraint anomaly_instance_db_name_fkey FOREIGN KEY (instance, db_name) references db(instance, name);
ALTER TABLE anomaly DROP COLUMN database_id;
CREATE UNIQUE INDEX idx_anomaly_unique_project_instance_dn_name_type ON anomaly(project, instance, db_name, type);

DROP INDEX IF EXISTS idx_deployment_config_unique_project_id;
ALTER TABLE deployment_config ADD COLUMN project TEXT;
UPDATE deployment_config SET project = project.resource_id FROM project WHERE project.id = deployment_config.project_id;
ALTER TABLE deployment_config DROP COLUMN project_id;
ALTER TABLE deployment_config ALTER COLUMN project SET NOT NULL;
ALTER TABLE deployment_config ADD constraint deployment_config_project_fkey FOREIGN KEY (project) references project(resource_id);
CREATE UNIQUE INDEX idx_deployment_config_unique_project ON deployment_config(project);

DROP INDEX IF EXISTS idx_worksheet_creator_id_project_id;
ALTER TABLE worksheet ADD COLUMN project TEXT;
UPDATE worksheet SET project = project.resource_id FROM project WHERE project.id = worksheet.project_id;
ALTER TABLE worksheet DROP COLUMN project_id;
ALTER TABLE worksheet ALTER COLUMN project SET NOT NULL;
ALTER TABLE worksheet ADD constraint worksheet_project_fkey FOREIGN KEY (project) references project(resource_id);
CREATE INDEX idx_worksheet_creator_id_project ON worksheet(creator_id, project);

ALTER TABLE worksheet ADD COLUMN instance TEXT;
ALTER TABLE worksheet ADD COLUMN db_name TEXT;
UPDATE worksheet SET instance = db.instance, db_name = db.name FROM db WHERE worksheet.database_id IS NOT NULL AND db.id = worksheet.database_id;
ALTER TABLE worksheet DROP COLUMN database_id;

ALTER TABLE slow_query ADD COLUMN instance TEXT;
ALTER TABLE slow_query ADD COLUMN db_name TEXT;
UPDATE slow_query SET db_name = db.name FROM db WHERE slow_query.database_id IS NOT NULL AND db.id = slow_query.database_id;
UPDATE slow_query SET instance = instance.resource_id FROM instance WHERE instance.id = slow_query.instance_id;
ALTER TABLE slow_query ALTER COLUMN instance SET NOT NULL;
ALTER TABLE slow_query ADD constraint slow_query_instance_fkey FOREIGN KEY (instance) references instance(resource_id);

DROP INDEX IF EXISTS idx_db_group_unique_project_id_resource_id;
DROP INDEX IF EXISTS idx_db_group_unique_project_id_placeholder;
ALTER TABLE db_group ADD COLUMN project TEXT;
UPDATE db_group SET project = project.resource_id FROM project WHERE project.id = db_group.project_id;
ALTER TABLE db_group DROP COLUMN project_id;
ALTER TABLE db_group ALTER COLUMN project SET NOT NULL;
ALTER TABLE db_group ADD constraint db_group_project_fkey FOREIGN KEY (project) references project(resource_id);
CREATE UNIQUE INDEX idx_db_group_unique_project_resource_id ON db_group(project, resource_id);
CREATE UNIQUE INDEX idx_db_group_unique_project_placeholder ON db_group(project, placeholder);

DROP INDEX IF EXISTS idx_changelist_project_id_name;
ALTER TABLE changelist ADD COLUMN project TEXT;
UPDATE changelist SET project = project.resource_id FROM project WHERE project.id = changelist.project_id;
ALTER TABLE changelist DROP COLUMN project_id;
ALTER TABLE changelist ALTER COLUMN project SET NOT NULL;
ALTER TABLE changelist ADD constraint changelist_project_fkey FOREIGN KEY (project) references project(resource_id);
CREATE UNIQUE INDEX idx_changelist_project_name ON changelist(project, name);

DROP INDEX IF EXISTS idx_revision_unique_database_id_version_deleted_at_null;
DROP INDEX IF EXISTS idx_revision_database_id_version;
ALTER TABLE revision ADD COLUMN instance TEXT;
ALTER TABLE revision ADD COLUMN db_name TEXT;
UPDATE revision SET instance = db.instance, db_name = db.name FROM db WHERE db.id = revision.database_id;
ALTER TABLE revision ALTER COLUMN instance SET NOT NULL;
ALTER TABLE revision ALTER COLUMN db_name SET NOT NULL;
ALTER TABLE revision ADD constraint revision_instance_db_name_fkey FOREIGN KEY (instance, db_name) references db(instance, name);
ALTER TABLE revision DROP COLUMN database_id;
CREATE UNIQUE INDEX IF NOT EXISTS idx_revision_unique_instance_db_name_version_deleted_at_null ON revision(instance, db_name, version) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_revision_instance_db_name_version ON revision(instance, db_name, version);

DROP INDEX IF EXISTS idx_sync_history_database_id_created_at;
ALTER TABLE sync_history ADD COLUMN instance TEXT;
ALTER TABLE sync_history ADD COLUMN db_name TEXT;
UPDATE sync_history SET instance = db.instance, db_name = db.name FROM db WHERE db.id = sync_history.database_id;
ALTER TABLE sync_history ALTER COLUMN instance SET NOT NULL;
ALTER TABLE sync_history ALTER COLUMN db_name SET NOT NULL;
ALTER TABLE sync_history ADD constraint sync_history_instance_db_name_fkey FOREIGN KEY (instance, db_name) references db(instance, name);
ALTER TABLE sync_history DROP COLUMN database_id;
CREATE INDEX IF NOT EXISTS idx_sync_history_instance_db_name_created_at ON sync_history (instance, db_name, created_at);

DROP INDEX IF EXISTS idx_changelog_database_id;
ALTER TABLE changelog ADD COLUMN instance TEXT;
ALTER TABLE changelog ADD COLUMN db_name TEXT;
UPDATE changelog SET instance = db.instance, db_name = db.name FROM db WHERE db.id = changelog.database_id;
ALTER TABLE changelog ALTER COLUMN instance SET NOT NULL;
ALTER TABLE changelog ALTER COLUMN db_name SET NOT NULL;
ALTER TABLE changelog ADD constraint changelog_instance_db_name_fkey FOREIGN KEY (instance, db_name) references db(instance, name);
ALTER TABLE changelog DROP COLUMN database_id;
CREATE INDEX IF NOT EXISTS idx_changelog_instance_db_name ON changelog (instance, db_name);

