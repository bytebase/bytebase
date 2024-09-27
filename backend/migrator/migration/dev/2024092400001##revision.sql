CREATE TABLE IF NOT EXISTS revision (
    id BIGSERIAL PRIMARY KEY,
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    database_id INTEGER NOT NULL REFERENCES db (id),
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    payload JSONB NOT NULL DEFAULT '{}'
);