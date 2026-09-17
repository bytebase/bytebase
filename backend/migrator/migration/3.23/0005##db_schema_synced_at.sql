ALTER TABLE db_schema ADD COLUMN synced_at timestamptz NOT NULL DEFAULT '-infinity';
