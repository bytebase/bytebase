CREATE TABLE migration_history (
    id STRING(36) NOT NULL,
    created_by STRING(MAX) NOT NULL,
    created_ts INT64 NOT NULL,
    updated_by STRING(MAX) NOT NULL,
    updated_ts INT64 NOT NULL,
    release_version STRING(MAX) NOT NULL,
    namespace STRING(MAX) NOT NULL,
    sequence INT64 NOT NULL,
    CONSTRAINT sequence_is_non_negative CHECK (sequence >= 0),
    source STRING(MAX) NOT NULL,
    type STRING(MAX) NOT NULL,
    status STRING(MAX) NOT NULL,
    version STRING(MAX) NOT NULL,
    description STRING(MAX) NOT NULL,
    statement STRING(MAX) NOT NULL,
    schema STRING(MAX) NOT NULL,
    schema_prev STRING(MAX) NOT NULL,
    execution_duration_ns INT64 NOT NULL,
    issue_id STRING(MAX) NOT NULL,
    payload STRING(MAX) NOT NULL
) PRIMARY KEY(id);

CREATE UNIQUE INDEX bytebase_idx_unique_migration_history_namespace_sequence ON migration_history (namespace, sequence);

CREATE UNIQUE INDEX bytebase_idx_unique_migration_history_namespace_version ON migration_history (namespace, version);

CREATE INDEX bytebase_idx_migration_history_namespace_source_type ON migration_history(namespace, source, type);

CREATE INDEX bytebase_idx_migration_history_namespace_created ON migration_history(namespace, created_ts);