CREATE TABLE query_history (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    database_id INTEGER NULL REFERENCES db (id),
    statement TEXT NOT NULL,
    type TEXT NOT NULL, -- the history type, support QUERY and EXPORT.
    payload JSONB NOT NULL DEFAULT '{}' -- saved for details, like error, duration, etc.
);

CREATE INDEX idx_query_history_creator_id_project_id ON query_history(creator_id, project_id);

CREATE INDEX idx_query_history_created_ts ON query_history(created_ts);

ALTER SEQUENCE query_history_id_seq RESTART WITH 101;

CREATE TRIGGER update_query_history_updated_ts
BEFORE
UPDATE
    ON query_history FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
