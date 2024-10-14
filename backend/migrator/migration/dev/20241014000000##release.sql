CREATE TABLE IF NOT EXISTS revision (
    id BIGSERIAL PRIMARY KEY,
    database_id INTEGER NOT NULL REFERENCES db (id),
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts TIMESTAMPTZ NOT NULL DEFAULT now(),
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS release (
    id BIGSERIAL PRIMARY KEY,
    project_id INTEGER NOT NULL REFERENCES project (id),
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts TIMESTAMPTZ NOT NULL DEFAULT now(),
    payload JSONB NOT NULL DEFAULT '{}'
);
