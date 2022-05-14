-- extension stores the extensions for a particular database.
-- data is synced periodically from the instance.
CREATE TABLE extension (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    schema TEXT NOT NULL,
    description TEXT NOT NULL
);

CREATE INDEX idx_extension_database_id ON extension(database_id);

CREATE UNIQUE INDEX idx_extension_unique_database_id_name_schema ON extension(database_id, name, schema);

ALTER SEQUENCE extension_id_seq RESTART WITH 101;

CREATE TRIGGER update_extension_updated_ts
BEFORE
UPDATE
    ON extension FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
