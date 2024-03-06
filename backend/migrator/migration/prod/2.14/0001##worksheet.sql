CREATE TABLE worksheet (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    database_id INTEGER NULL REFERENCES db (id),
    name TEXT NOT NULL,
    statement TEXT NOT NULL,
    visibility TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_worksheet_creator_id ON worksheet(creator_id);

CREATE INDEX idx_worksheet_project_id ON worksheet(project_id);

CREATE TRIGGER update_worksheet_updated_ts
BEFORE
UPDATE
    ON worksheet FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
