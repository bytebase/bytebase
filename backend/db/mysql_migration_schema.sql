-- This is the bytebase schema to track migration info for MySQL
-- Create a database called bytebase
CREATE DATABASE bytebase CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_0900_ai_ci';

-- Create migration_history table
CREATE TABLE bytebase.migration_history (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    created_by TEXT NOT NULL,
    created_ts BIGINT NOT NULL,
    updated_by TEXT NOT NULL,
    updated_ts BIGINT NOT NULL,
    -- Allows granular tracking of migration history (e.g If an application manages schemas for a multi-tenant service and each tenant has its own schema, that application can use namespace to record the tenant name to track the per-tenant schema migration)
    -- Since bytebase also manages different application databases from an instance, it leverages this field to track each database migration history.
    namespace TEXT NOT NULL,
    -- Used to detect out of order migration together with 'namespace' and 'version' column.
    sequence INTEGER UNSIGNED NOT NULL,
    `type` ENUM('BASELINE', 'SQL', 'SQL_ROLLBACK', 'DELETED') NOT NULL,
    version TEXT NOT NULL,
    description TEXT NOT NULL,
    -- Recorded the migration statement
    statement TEXT NOT NULL,
    execution_duration INTEGER NOT NULL
);

CREATE UNIQUE INDEX bytebase_idx_unique_migration_history_namespace_sequence ON bytebase.migration_history (namespace(256), sequence);

CREATE UNIQUE INDEX bytebase_idx_unique_migration_history_namespace_version ON bytebase.migration_history (namespace(256), version(256));

CREATE INDEX bytebase_idx_migration_history_namespace_type ON bytebase.migration_history(namespace(256), `type`);

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