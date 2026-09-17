ALTER TABLE db_schema ADD COLUMN synced_at timestamptz NOT NULL DEFAULT to_timestamp(0);
