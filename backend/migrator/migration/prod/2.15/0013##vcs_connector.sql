ALTER TABLE repository ADD COLUMN payload JSONB NOT NULL DEFAULT '{}';
UPDATE repository SET payload = json_build_object('title', name, 'fullPath', full_path, 'webUrl', web_url, 'branch', branch, 'baseDirectory', base_directory, 'externalId', external_id, 'externalWebhookId', external_webhook_id, 'webhookSecretToken', webhook_secret_token);
ALTER TABLE repository DROP COLUMN name, DROP COLUMN full_path, DROP COLUMN web_url, DROP COLUMN branch, DROP COLUMN base_directory, DROP COLUMN external_id, DROP COLUMN external_webhook_id, DROP COLUMN webhook_url_host, DROP COLUMN webhook_endpoint_id, DROP COLUMN webhook_secret_token;
ALTER TABLE repository RENAME TO vcs_connector;
