CREATE TABLE sql_lint_config (
  id TEXT NOT NULL PRIMARY KEY,
  creator_id INTEGER NOT NULL REFERENCES principal (id),
  created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
  updater_id INTEGER NOT NULL REFERENCES principal (id),
  updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
  config JSONB NOT NULL DEFAULT '{}'
);