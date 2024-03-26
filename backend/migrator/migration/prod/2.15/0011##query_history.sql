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
    database TEXT NOT NULL,
    type TEXT NOT NULL, -- the history type, support QUERY and EXPORT.
    payload JSONB NOT NULL DEFAULT '{}' -- saved for details, like error, duration, etc.
);