-- This is the bytebase schema to track migration info for MySQL
-- Create a database called bytebase
CREATE DATABASE bytebase;

-- Create migration_history table
-- Note, we don't create trigger to update created_ts and updated_ts because that may causes error:
-- ERROR 1419 (HY000): You do not have the SUPER privilege and binary logging is enabled (you *might* want to use the less safe log_bin_trust_function_creators variable)
CREATE TABLE bytebase.migration_history (
    id INTEGER,
    created_by TEXT NOT NULL,
    created_ts BIGINT NOT NULL,
    updated_by TEXT NOT NULL,
    updated_ts BIGINT NOT NULL,
    -- Record the client version creating this migration history. For Bytebase, we use its binary release version. Different Bytebase release might
    -- record different history info and thie field helps to handle such situation properly. Moreover, it helps debugging.
    release_version TEXT NOT NULL,
    -- Allows granular tracking of migration history (e.g If an application manages schemas for a multi-tenant service and each tenant has its own schema, that application can use namespace to record the tenant name to track the per-tenant schema migration)
    -- Since bytebase also manages different application databases from an instance, it leverages this field to track each database migration history.
    namespace TEXT NOT NULL,
    -- Used to detect out of order migration together with 'namespace' and 'version' column.
    sequence INTEGER UNSIGNED NOT NULL,
    -- We call it engine because maybe we could load history from other migration tool.
    `engine` ENUM('UI' = 0, 'VCS' = 1) NOT NULL,
    `type` ENUM('BASELINE' = 0, 'MIGRATE' = 1, 'BRANCH' = 2, 'DATA' = 3) NOT NULL,
    -- MySQL runs DDL in its own transaction, so we can't record DDL and migration_history into a single transaction.
    -- Thus, we create a "PENDING" record before applying the DDL and update that record to "DONE" after applying the DDL.
    `status` ENUM('PENDING' = 0, 'DONE' = 1, 'FAILED' = 2) NOT NULL,
    -- Record the migration version.
    version TEXT NOT NULL,
    description TEXT NOT NULL,
    -- Record the migration statement
    statement TEXT NOT NULL,
    -- Record the schema after migration
    `schema` TEXT NOT NULL,
    -- Record the schema before migration. Though we could also fetch it from the previous migration history, it would complicate fetching logic.
    -- Besides, by storing the schema_prev, we can perform consistency check to see if the migration history has any gaps.
    schema_prev TEXT NOT NULL,
    execution_duration INTEGER NOT NULL,
    issue_id TEXT NOT NULL,
    payload TEXT NOT NULL
) ENGINE = MergeTree()
PRIMARY KEY id;
