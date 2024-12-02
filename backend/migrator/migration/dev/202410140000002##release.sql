CREATE TABLE IF NOT EXISTS revision (
    id BIGSERIAL PRIMARY KEY,
    database_id INTEGER NOT NULL REFERENCES db (id),
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleter_id INTEGER REFERENCES principal (id),
    deleted_ts TIMESTAMPTZ,
    version TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_revision_unique_database_id_version_deleted_ts_null ON revision (database_id, version) WHERE deleted_ts IS NULL;

CREATE INDEX IF NOT EXISTS idx_revision_database_id_version ON revision (database_id, version);

CREATE TABLE IF NOT EXISTS release (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    project_id INTEGER NOT NULL REFERENCES project (id),
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts TIMESTAMPTZ NOT NULL DEFAULT now(),
    payload JSONB NOT NULL DEFAULT '{}'
);
