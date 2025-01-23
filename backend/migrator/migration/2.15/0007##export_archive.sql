CREATE TABLE export_archive (
  id SERIAL PRIMARY KEY,
  created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
  bytes BYTEA,
  payload JSONB NOT NULL DEFAULT '{}'
);