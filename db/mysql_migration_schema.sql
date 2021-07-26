-- This is the bytebase schema to track migration info for MySQL
-- Create a database called bytebase
CREATE DATABASE bytebase CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_general_ci';

CREATE TABLE bytebase.setting (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    created_by TEXT NOT NULL,
    created_ts BIGINT NOT NULL,
    updated_by TEXT NOT NULL,
    updated_ts BIGINT NOT NULL,
    name TEXT NOT NULL,
    value TEXT NOT NULL,
    description TEXT NOT NULL
);

CREATE UNIQUE INDEX bytebase_idx_unique_setting_name ON bytebase.setting (name(256));

-- Insert schema version 1
INSERT INTO
    bytebase.setting (
        created_by,
        created_ts,
        updated_by,
        updated_ts,
        name,
        value,
        description
    )
VALUES
    (
        'bytebase',
        UNIX_TIMESTAMP(),
        'bytebase',
        UNIX_TIMESTAMP(),
        'bb.schema.version',
        '1',
        'Schema version'
    );

-- Create migration_history table
-- Note, we don't create trigger to update created_ts and updated_ts because that may causes error:
-- ERROR 1419 (HY000): You do not have the SUPER privilege and binary logging is enabled (you *might* want to use the less safe log_bin_trust_function_creators variable)
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
    -- We call it engine because maybe we could load history from other migration tool.
    `engine` ENUM('UI', 'VCS') NOT NULL,
    `type` ENUM('BASELINE', 'SQL') NOT NULL,
    version TEXT NOT NULL,
    description TEXT NOT NULL,
    -- Recorded the migration statement
    statement TEXT NOT NULL,
    execution_duration INTEGER NOT NULL,
    issue_id TEXT NOT NULL,
    payload TEXT NOT NULL
);

CREATE UNIQUE INDEX bytebase_idx_unique_migration_history_namespace_sequence ON bytebase.migration_history (namespace(256), sequence);

CREATE UNIQUE INDEX bytebase_idx_unique_migration_history_namespace_engine_version ON bytebase.migration_history (namespace(256), `engine`, version(256));

CREATE INDEX bytebase_idx_migration_history_namespace_engine_type ON bytebase.migration_history(namespace(256), `engine`, `type`);

CREATE INDEX bytebase_idx_migration_history_namespace_created ON bytebase.migration_history(namespace(256), `created_ts`);