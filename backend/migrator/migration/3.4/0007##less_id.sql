ALTER TABLE stage ADD COLUMN environment TEXT;
UPDATE stage SET environment = environment.resource_id FROM environment WHERE environment.id = stage.environment_id;
ALTER TABLE stage DROP COLUMN environment_id;
ALTER TABLE stage ALTER COLUMN environment SET NOT NULL;

