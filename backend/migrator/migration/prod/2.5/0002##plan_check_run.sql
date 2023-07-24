CREATE TABLE plan_check_run (
    id SERIAL PRIMARY KEY,
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    plan_id INTEGER NOT NULL REFERENCES plan (id),
    status TEXT NOT NULL CHECK (status IN ('RUNNING', 'DONE', 'FAILED', 'CANCELED')),
    type TEXT NOT NULL CHECK (type LIKE 'bb.plan-check.%'),
    config JSONB NOT NULL DEFAULT '{}',
    result JSONB NOT NULL DEFAULT '{}',
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_plan_check_run_plan_id ON plan_check_run (plan_id);

ALTER SEQUENCE plan_check_run_id_seq RESTART WITH 101;

CREATE TRIGGER update_plan_check_run_updated_ts
BEFORE
UPDATE
    ON plan_check_run FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
