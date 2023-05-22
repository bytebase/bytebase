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
    workflow JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_plan_project_id ON plan(project_id);

ALTER SEQUENCE plan_id_seq RESTART WITH 101;

CREATE TRIGGER update_plan_updated_ts
BEFORE
UPDATE
    ON plan FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

CREATE TABLE plan_check_run (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    plan_id BIGINT NOT NULL REFERENCES plan (id),
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    result JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_plan_check_run_plan_id ON plan_check_run(plan_id);

ALTER SEQUENCE plan_check_run_id_seq RESTART WITH 101;

CREATE TRIGGER update_plan_check_run_updated_ts
BEFORE
UPDATE
    ON plan_check_run FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
