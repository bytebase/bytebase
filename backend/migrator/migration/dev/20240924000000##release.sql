CREATE TABLE release (
    id BIGSERIAL PRIMARY KEY,
    project_id INTEGER NOT NULL REFERENCES project (id),
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    payload JSONB NOT NULL DEFAULT '{}'
);