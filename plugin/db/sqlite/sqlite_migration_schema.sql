-- Create migration_history table
CREATE TABLE bytebase_migration_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_by TEXT NOT NULL,
    created_ts INTEGER NOT NULL,
    updated_by TEXT NOT NULL,
    updated_ts INTEGER NOT NULL,
    -- Record the client version creating this migration history. For Bytebase, we use its binary release version. Different Bytebase release might
    -- record different history info and thie field helps to handle such situation properly. Moreover, it helps debugging.
    release_version TEXT NOT NULL,
    -- Allows granular tracking of migration history (e.g If an application manages schemas for a multi-tenant service and each tenant has its own schema, that application can use namespace to record the tenant name to track the per-tenant schema migration)
    -- Since bytebase also manages different application databases from an instance, it leverages this field to track each database migration history.
    namespace TEXT NOT NULL,
    -- Used to detect out of order migration together with 'namespace' and 'version' column.
    sequence INTEGER UNSIGNED NOT NULL,
    -- We call it source because maybe we could load history from other migration tool.
    -- Current allowed values are UI, VCS, LIBRARY.
    source TEXT NOT NULL,
    -- Current allowed values are BASELINE, MIGRATE, BRANCH, DATA.
    type TEXT NOT NULL,
    -- Current allowed values are PENDING, DONE, FAILED.
    -- We create a "PENDING" record before applying the DDL and update that record to "DONE" after applying the DDL.
    status TEXT NOT NULL,
    -- Record the migration version.
    version TEXT NOT NULL,
    description TEXT NOT NULL,
    -- Record the migration statement
    statement TEXT NOT NULL,
    -- Record the schema after migration
    schema MEDIUMTEXT NOT NULL,
    -- Record the schema before migration. Though we could also fetch it from the previous migration history, it would complicate fetching logic.
    -- Besides, by storing the schema_prev, we can perform consistency check to see if the migration history has any gaps.
    schema_prev MEDIUMTEXT NOT NULL,
    execution_duration_ns INTEGER NOT NULL,
    issue_id TEXT NOT NULL,
    payload TEXT NOT NULL
);

CREATE UNIQUE INDEX bytebase_idx_unique_migration_history_namespace_sequence ON bytebase_migration_history (namespace, sequence);

CREATE UNIQUE INDEX bytebase_idx_unique_migration_history_namespace_version ON bytebase_migration_history (namespace, version);

CREATE INDEX bytebase_idx_migration_history_namespace_source_type ON bytebase_migration_history(namespace, source, type);

CREATE INDEX bytebase_idx_migration_history_namespace_created ON bytebase_migration_history(namespace, created_ts);
