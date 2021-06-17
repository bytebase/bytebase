-- This is the bytebase schema to track migration info for MySQL
-- Create a database called bytebase
CREATE DATABASE bytebase CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci';

-- Create migration_history table
CREATE TABLE bytebase.migration_history (
    -- Can be used to detect out of order migration together with 'version' column.
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    created_by TEXT NOT NULL,
    created_ts BIGINT NOT NULL,
    updated_by TEXT NOT NULL,
    updated_ts BIGINT NOT NULL,
    -- Allows granular tracking of migration history (e.g If an application manages schemas for a multi-tenant service and each tenant has its own schema, that application can use namespace to record the tenant name to track the per-tenant schema migration)
    -- Since bytebase also manages different application databases from an instance, it leverages this field to track different databases using bytebase.{{dbname}} format.
    namespace TEXT NOT NULL,
    `type` ENUM('BASELINE', "SQL", "SQL_ROLLBACK", "DELETED") NOT NULL,
    version TEXT NOT NULL,
    description TEXT NOT NULL,
    -- Recorded the migration sql
    `sql` TEXT NOT NULL,
    execution_duration INTEGER NOT NULL,
    execution_status ENUM('RUNNING', 'DONE', 'FAILED') NOT NULL,
    execution_detail TEXT NOT NULL
);

CREATE UNIQUE INDEX bytebase_idx_unique_migration_history_version ON bytebase.migration_history (version(256));

CREATE INDEX bytebase_idx_migration_history_namespace ON bytebase.migration_history(namespace(256));

CREATE TRIGGER bytebase.trigger_update_migration_history_creation_time BEFORE
INSERT
    ON bytebase.migration_history FOR each ROW BEGIN
SET
    new.created_ts = unix_timestamp();

SET
    new.updated_ts = unix_timestamp();

END;

CREATE TRIGGER bytebase.trigger_update_migration_history_modification_time BEFORE
UPDATE
    ON bytebase.migration_history FOR each ROW BEGIN
SET
    new.updated_ts = unix_timestamp();

END;