CREATE TABLE plan (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_plan_project_id ON plan(project_id);

ALTER SEQUENCE plan_id_seq RESTART WITH 101;

CREATE TRIGGER update_plan_updated_ts
BEFORE
UPDATE
    ON plan FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
