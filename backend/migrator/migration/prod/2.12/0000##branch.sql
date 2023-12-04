CREATE TABLE branch (
  id SERIAL PRIMARY KEY,
  row_status row_status NOT NULL DEFAULT 'NORMAL',
  creator_id INTEGER NOT NULL REFERENCES principal (id),
  created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
  updater_id INTEGER NOT NULL REFERENCES principal (id),
  updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
  project_id INTEGER NOT NULL REFERENCES project (id),
  name TEXT NOT NULL,
  engine TEXT NOT NULL,
  base JSONB NOT NULL DEFAULT '{}',
  head JSONB NOT NULL DEFAULT '{}',
  config JSONB NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_branch_unique_project_id_name ON branch(project_id, name);

ALTER SEQUENCE branch_id_seq RESTART WITH 101;

CREATE TRIGGER update_branch_updated_ts
BEFORE
UPDATE
    ON branch FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();