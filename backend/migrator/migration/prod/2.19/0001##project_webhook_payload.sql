ALTER TABLE project_webhook ADD COLUMN payload JSONB NOT NULL DEFAULT '{}';
