CREATE TABLE revision (
    id BIGSERIAL PRIMARY KEY,
    database_id INTEGER NOT NULL REFERENCES db (id),
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleter_id INTEGER REFERENCES principal (id),
    deleted_ts TIMESTAMPTZ,
    version TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'
);

ALTER SEQUENCE revision_id_seq RESTART WITH 101;

CREATE UNIQUE INDEX IF NOT EXISTS idx_revision_unique_database_id_version_deleted_ts_null ON revision (database_id, version) WHERE deleted_ts IS NULL;

CREATE INDEX IF NOT EXISTS idx_revision_database_id_version ON revision (database_id, version);

CREATE TABLE sync_history (
    id BIGSERIAL PRIMARY KEY,
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts TIMESTAMPTZ NOT NULL DEFAULT now(),
    database_id INTEGER NOT NULL REFERENCES db (id),
    metadata JSON NOT NULL DEFAULT '{}',
    raw_dump TEXT NOT NULL DEFAULT ''
);

ALTER SEQUENCE sync_history_id_seq RESTART WITH 101;

CREATE INDEX IF NOT EXISTS idx_sync_history_database_id_created_ts ON sync_history (database_id, created_ts);

CREATE TABLE changelog (
    id BIGSERIAL PRIMARY KEY,
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts TIMESTAMPTZ NOT NULL DEFAULT now(),
    database_id INTEGER NOT NULL REFERENCES db (id),
    status TEXT NOT NULL CONSTRAINT changelog_status_check CHECK (status IN ('PENDING', 'DONE', 'FAILED')),
    prev_sync_history_id BIGINT REFERENCES sync_history (id),
    sync_history_id BIGINT REFERENCES sync_history (id),
    payload JSONB NOT NULL DEFAULT '{}'
);

ALTER SEQUENCE changelog_id_seq RESTART WITH 101;

CREATE INDEX IF NOT EXISTS idx_changelog_database_id ON changelog (database_id);
