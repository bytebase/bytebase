ALTER TABLE instance ADD COLUMN environment TEXT REFERENCES environment (resource_id);
UPDATE instance SET environment = (SELECT resource_id FROM environment WHERE instance.environment_id = environment.id);
ALTER TABLE instance DROP COLUMN environment_id;