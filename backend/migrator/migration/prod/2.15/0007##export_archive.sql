CREATE TABLE export_archive (
  id SERIAL PRIMARY KEY,
  row_status row_status NOT NULL DEFAULT 'NORMAL',
  created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
  updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
  bytes BYTEA,
  payload JSONB NOT NULL DEFAULT '{}'
);