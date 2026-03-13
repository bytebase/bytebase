-- Drop unused `id` column from project_webhook, use `resource_id` as PK.
-- `id` has no FK references; all lookups use `resource_id`.

ALTER TABLE project_webhook DROP CONSTRAINT IF EXISTS project_webhook_pkey;
DROP INDEX IF EXISTS idx_project_webhook_unique_resource_id;

ALTER TABLE project_webhook DROP COLUMN IF EXISTS id;

ALTER TABLE project_webhook ADD CONSTRAINT project_webhook_pkey PRIMARY KEY (resource_id) NOT VALID;

DROP SEQUENCE IF EXISTS project_webhook_id_seq;
