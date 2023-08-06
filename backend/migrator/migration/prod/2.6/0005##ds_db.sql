DROP INDEX IF EXISTS idx_data_source_instance_id;
DROP INDEX IF EXISTS idx_data_source_unique_database_id_name;
ALTER TABLE data_source DROP COLUMN database_id;
CREATE UNIQUE INDEX idx_data_source_unique_instance_id_name ON data_source(instance_id, name);
DELETE FROM db WHERE name = '*';