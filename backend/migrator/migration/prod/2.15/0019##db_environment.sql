ALTER TABLE db ADD COLUMN environment TEXT REFERENCES environment (resource_id);
UPDATE db SET environment = (SELECT resource_id FROM environment WHERE db.environment_id = environment.id) WHERE db.environment_id IS NOT NULL;
ALTER TABLE db DROP COLUMN environment_id;