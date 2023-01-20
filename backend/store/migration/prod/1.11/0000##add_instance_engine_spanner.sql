ALTER TABLE instance DROP CONSTRAINT instance_engine_check;

ALTER TABLE instance ADD CONSTRAINT instance_engine_check CHECK (engine IN ('MYSQL', 'POSTGRES', 'TIDB', 'CLICKHOUSE', 'SNOWFLAKE', 'SQLITE', 'MONGODB', 'SPANNER'));
