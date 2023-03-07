-- instance change history records the changes an instance and its databases.
CREATE TABLE instance_change_history (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    -- NULL means an instance-level change.
    database_id INTEGER REFERENCES db (id),
    -- issue_id is nullable because this field is backfilled and may not be present.
    issue_id INTEGER REFERENCES issue (id),
    -- Record the client version creating this change history. For Bytebase, we use its binary release version. Different Bytebase release might
    -- record different history info and this field helps to handle such situation properly. Moreover, it helps debugging.
    release_version TEXT NOT NULL,
    -- Used to detect out of order change history together with 'namespace' and 'version' column.
    sequence BIGINT NOT NULL CONSTRAINT instance_change_history_sequence_check CHECK (sequence >= 0),
    -- We call it source because maybe we could load history from other migration tool.
    -- Currently allowed values are UI, VCS, LIBRARY.
    source TEXT NOT NULL CONSTRAINT instance_change_history_source_check CHECK (source IN ('UI', 'VCS', 'LIBRARY')),
    -- Currently allowed values are BASELINE, MIGRATE, MIGRATE_SDL, BRANCH, DATA.
    type TEXT NOT NULL CONSTRAINT instance_change_history_type_check CHECK (type IN ('BASELINE', 'MIGRATE', 'MIGRATE_SDL', 'BRANCH', 'DATA')),
    -- Currently allowed values are PENDING, DONE, FAILED.
    -- PostgreSQL can't do cross database transaction, so we can't record DDL and change_history into a single transaction.
    -- Thus, we create a "PENDING" record before applying the DDL and update that record to "DONE" after applying the DDL.
    status TEXT NOT NULL CONSTRAINT instance_change_history_status_check CHECK (status IN ('PENDING', 'DONE', 'FAILED')),
    -- Record the change version.
    version TEXT NOT NULL,
    description TEXT NOT NULL,
    -- Record the change statement
    statement TEXT NOT NULL,
    -- Record the schema after change
    schema TEXT NOT NULL,
    -- Record the schema before change. Though we could also fetch it from the previous change history, it would complicate fetching logic.
    -- Besides, by storing the schema_prev, we can perform consistency check to see if the change history has any gaps.
    schema_prev TEXT NOT NULL,
    execution_duration_ns BIGINT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_instance_change_history_unique_instance_id_database_id_sequence ON instance_change_history (instance_id, database_id, sequence);

CREATE UNIQUE INDEX idx_instance_change_history_unique_instance_id_database_id_version ON instance_change_history (instance_id, database_id, version);

ALTER SEQUENCE instance_change_history_id_seq RESTART WITH 101;

CREATE TRIGGER update_instance_change_history_updated_ts
BEFORE
UPDATE
    ON instance_change_history FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

