ALTER TABLE slow_query DROP COLUMN instance_id;
ALTER TABLE slow_query DROP COLUMN database_id;
ALTER TABLE anomaly DROP COLUMN instance_id;
ALTER TABLE worksheet DROP COLUMN IF EXISTS database_id;