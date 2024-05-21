CREATE TABLE task_run_log (
    id BIGSERIAL PRIMARY KEY,
    task_run_id INTEGER NOT NULL REFERENCES task_run (id),
    created_ts TIMESTAMP NOT NULL DEFAULT now(),
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_task_run_log_task_run_id ON task_run_log(task_run_id);

ALTER SEQUENCE task_run_log_id_seq RESTART WITH 101;
