-- task_dag describes task dependency relationship
-- from_task_id is blocked by to_task_id
CREATE TABLE task_dag (
    id SERIAL PRIMARY KEY,
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    from_task_id INTEGER NOT NULL REFERENCES task (id),
    to_task_id INTEGER NOT NULL REFERENCES task (id),
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_task_dag_from_task_id ON task_dag(from_task_id);

CREATE INDEX idx_task_dag_to_task_id ON task_dag(to_task_id);

ALTER SEQUENCE task_dag_id_seq RESTART WITH 101;

CREATE TRIGGER update_task_dag_updated_ts
BEFORE
UPDATE
    ON task_dag FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
