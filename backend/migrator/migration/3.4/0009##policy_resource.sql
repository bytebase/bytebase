DROP INDEX IF EXISTS idx_policy_unique_resource_type_resource_id_type;
ALTER TABLE policy ADD COLUMN IF NOT EXISTS resource TEXT;
-- DML.
UPDATE policy SET resource = '' WHERE resource_type = 'WORKSPACE';
UPDATE policy SET resource = 'projectss/' || project.resource_id FROM project WHERE resource_type = 'PROJECT' AND project.id = policy.resource_id;
UPDATE policy SET resource = 'environments/' || environment.resource_id FROM environment WHERE resource_type = 'ENVIRONMENT' AND environment.id = policy.resource_id;
UPDATE policy SET resource = 'instances/' || instance.resource_id FROM instance WHERE resource_type = 'INSTANCE' AND instance.id = policy.resource_id;
UPDATE policy SET resource = 'instances/' || db.instance || '/databases/' || db.name FROM db WHERE resource_type = 'DATABASE' AND db.id = policy.resource_id;
ALTER TABLE policy DROP COLUMN IF EXISTS resource_id;
ALTER TABLE policy ALTER COLUMN resource SET NOT NULL;
CREATE UNIQUE INDEX idx_policy_unique_resource_type_resource_type ON policy(resource_type, resource, type);
