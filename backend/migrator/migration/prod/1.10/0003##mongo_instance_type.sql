-- NOTE: we did not declare a name for this constraint first, so this may not work.
ALTER TABLE instance DROP CONSTRAINT instance_engine_check;

ALTER TABLE instance ADD CONSTRAINT instance_engine_check CHECK (engine IN ('MYSQL', 'POSTGRES', 'TIDB', 'CLICKHOUSE', 'SNOWFLAKE', 'SQLITE', 'MONGODB'));