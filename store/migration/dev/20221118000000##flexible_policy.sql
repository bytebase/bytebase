CREATE TYPE resource_type AS ENUM ('WORKSPACE', 'ENVIRONMENT', 'PROJECT', 'INSTANCE', 'DATABASE');

DROP INDEX idx_policy_environment_id;
DROP INDEX idx_policy_unique_environment_id_type;

ALTER TABLE policy ADD COLUMN resource_type resource_type;
ALTER TABLE policy ADD COLUMN resource_id INTEGER;
ALTER TABLE policy ADD COLUMN inherit_from_parent BOOLEAN DEFAULT TRUE;
ALTER TABLE policy DROP CONSTRAINT policy_environment_id_fkey;

UPDATE policy SET resource_type='ENVIRONMENT', resource_id=environment_id;

ALTER TABLE policy ALTER COLUMN resource_type SET NOT NULL;
ALTER TABLE policy ALTER COLUMN resource_id SET NOT NULL;
ALTER TABLE policy DROP COLUMN environment_id;

CREATE UNIQUE INDEX idx_policy_unique_resource_type_resource_id_type ON policy(resource_type, resource_id, type);
