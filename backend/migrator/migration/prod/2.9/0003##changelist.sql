CREATE TABLE changelist (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    name TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_changelist_project_id_name ON changelist(project_id, name);

ALTER SEQUENCE changelist_id_seq RESTART WITH 101;

CREATE TRIGGER update_changelist_updated_ts
BEFORE
UPDATE
    ON changelist FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();