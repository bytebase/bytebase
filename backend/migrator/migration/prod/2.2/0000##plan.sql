CREATE TABLE plan (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    pipeline_id INTEGER REFERENCES pipeline (id),
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_plan_project_id ON plan(project_id);

CREATE INDEX idx_plan_pipeline_id ON plan(pipeline_id);

ALTER SEQUENCE plan_id_seq RESTART WITH 101;

CREATE TRIGGER update_plan_updated_ts
BEFORE
UPDATE
    ON plan FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

ALTER TABLE issue ADD COLUMN plan_id BIGINT REFERENCES plan (id);

CREATE INDEX idx_issue_plan_id ON issue (plan_id);