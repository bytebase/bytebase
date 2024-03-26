CREATE TABLE query_history (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id TEXT NOT NULL, -- the project resource id
    database TEXT NOT NULL, -- the database resource name, for example, instances/{instance}/databases/{database}
    statement TEXT NOT NULL,
    type TEXT NOT NULL, -- the history type, support QUERY and EXPORT.
    payload JSONB NOT NULL DEFAULT '{}' -- saved for details, like error, duration, etc.
);

CREATE INDEX idx_query_history_creator_id_database ON query_history(creator_id, database);

CREATE INDEX idx_query_history_created_ts ON query_history(created_ts);

ALTER SEQUENCE query_history_id_seq RESTART WITH 101;

CREATE TRIGGER update_query_history_updated_ts
BEFORE
UPDATE
    ON query_history FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
