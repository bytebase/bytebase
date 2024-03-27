ALTER TABLE repository ADD COLUMN payload JSONB NOT NULL DEFAULT '{}';
UPDATE repository SET payload = json_build_object('title', name, 'fullPath', full_path, 'webUrl', web_url, 'branch', branch, 'baseDirectory', base_directory, 'externalId', external_id, 'externalWebhookId', external_webhook_id, 'webhookSecretToken', webhook_secret_token);
ALTER TABLE repository DROP COLUMN name, DROP COLUMN full_path, DROP COLUMN web_url, DROP COLUMN branch, DROP COLUMN base_directory, DROP COLUMN external_id, DROP COLUMN external_webhook_id, DROP COLUMN webhook_url_host, DROP COLUMN webhook_endpoint_id, DROP COLUMN webhook_secret_token;

ALTER TABLE repository RENAME TO vcs_connector;

ALTER INDEX idx_repository_unique_project_id_resource_id RENAME TO idx_vcs_connector_unique_project_id_resource_id;
ALTER INDEX repository_pkey RENAME TO vcs_connector_pkey;

ALTER TABLE vcs_connector RENAME CONSTRAINT repository_creator_id_fkey TO vcs_connector_creator_id_fkey;
ALTER TABLE vcs_connector RENAME CONSTRAINT repository_project_id_fkey TO vcs_connector_project_id_fkey;
ALTER TABLE vcs_connector RENAME CONSTRAINT repository_updater_id_fkey TO vcs_connector_updater_id_fkey;
ALTER TABLE vcs_connector RENAME CONSTRAINT repository_vcs_id_fkey TO vcs_connector_vcs_id_fkey;

ALTER TRIGGER update_repository_updated_ts ON vcs_connector RENAME TO update_vcs_connector_updated_ts;
