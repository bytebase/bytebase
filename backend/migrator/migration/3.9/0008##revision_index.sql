DROP INDEX IF EXISTS idx_revision_unique_instance_db_name_version_deleted_at_null;
DROP INDEX IF EXISTS idx_revision_instance_db_name_version;

CREATE UNIQUE INDEX idx_revision_unique_instance_db_name_type_version_deleted_at_null ON revision(instance, db_name, (payload->>'type'), version) WHERE deleted_at IS NULL;
CREATE INDEX idx_revision_instance_db_name_type_version ON revision(instance, db_name, (payload->>'type'), version);