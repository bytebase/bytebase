CREATE TABLE db_schema (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id) ON DELETE CASCADE,
    metadata JSONB NOT NULL DEFAULT '{}',
    raw_dump TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX idx_db_schema_unique_database_id ON tbl(database_id);

ALTER SEQUENCE db_schema_id_seq RESTART WITH 101;

CREATE TRIGGER update_db_schema_updated_ts
BEFORE
UPDATE
    ON db_schema FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
