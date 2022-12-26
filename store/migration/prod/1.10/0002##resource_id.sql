ALTER TABLE environment ADD COLUMN resource_id TEXT;
ALTER TABLE project ADD COLUMN resource_id TEXT;
ALTER TABLE instance ADD COLUMN resource_id TEXT;

UPDATE environment SET resource_id=LOWER(name);
UPDATE project SET resource_id=CONCAT('project-', LEFT(gen_random_uuid()::TEXT, 8));
UPDATE instance SET resource_id=CONCAT('instance-', LEFT(gen_random_uuid()::TEXT, 8));

ALTER TABLE environment ALTER COLUMN resource_id SET NOT NULL;
ALTER TABLE project ALTER COLUMN resource_id SET NOT NULL;
ALTER TABLE instance ALTER COLUMN resource_id SET NOT NULL;

CREATE UNIQUE INDEX idx_environment_unique_resource_id ON environment(resource_id);
CREATE UNIQUE INDEX idx_project_unique_resource_id ON project(resource_id);
CREATE UNIQUE INDEX idx_instance_unique_resource_id ON instance(resource_id);
