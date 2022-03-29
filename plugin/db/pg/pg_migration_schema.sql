-- This is the bytebase schema to track migration info for Postgres
-- Create a database called bytebase in the driver.
-- CREATE DATABASE bytebase;

-- Create migration_history table
-- Note, we don't create trigger to update created_ts and updated_ts because that may causes error:
-- ERROR 1419 (HY000): You do not have the SUPER privilege and binary logging is enabled (you *might* want to use the less safe log_bin_trust_function_creators variable)
CREATE TABLE migration_history (
    id BIGSERIAL PRIMARY KEY,
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
    sequence BIGINT NOT NULL CHECK (sequence >= 0),
    -- We call it source because maybe we could load history from other migration tool.
    -- Current allowed values are UI, VCS, LIBRARY.
    source TEXT NOT NULL,
    -- Current allowed values are BASELINE, MIGRATE, BRANCH, DATA.
    type TEXT NOT NULL,
    -- Current allowed values are PENDING, DONE, FAILED.
    -- PostgreSQL can't do cross database transaction, so we can't record DDL and migration_history into a single transaction.
    -- Thus, we create a "PENDING" record before applying the DDL and update that record to "DONE" after applying the DDL.
    status TEXT NOT NULL,
    -- Record the migration version.
    version TEXT NOT NULL,
    description TEXT NOT NULL,
    -- Record the migration statement
    statement TEXT NOT NULL,
    -- Record the schema after migration
    schema TEXT NOT NULL,
    -- Record the schema before migration. Though we could also fetch it from the previous migration history, it would complicate fetching logic.
    -- Besides, by storing the schema_prev, we can perform consistency check to see if the migration history has any gaps.
    schema_prev TEXT NOT NULL,
    execution_duration_ns BIGINT NOT NULL,
    issue_id TEXT NOT NULL,
    payload TEXT NOT NULL
);

CREATE UNIQUE INDEX bytebase_idx_unique_migration_history_namespace_sequence ON migration_history (namespace, sequence);

CREATE UNIQUE INDEX bytebase_idx_unique_migration_history_namespace_version ON migration_history (namespace, version);

CREATE INDEX bytebase_idx_migration_history_namespace_source_type ON migration_history(namespace, source, type);

CREATE INDEX bytebase_idx_migration_history_namespace_created ON migration_history(namespace, created_ts);
