ALTER TABLE db_schema ADD COLUMN synced_at timestamptz;

-- Rows that predate the column have no record of when their dump was read.
-- db.metadata's lastSyncTime is the closest available value: when the database
-- last finished a sync, so later than the dump's read by that sync's duration,
-- and from a different sync than the one that wrote the dump where the two
-- raced. It is a past timestamp at migration time, so every read after the
-- upgrade is still newer and stores normally. A database that never recorded a
-- sync keeps NULL, which every sync is allowed to overwrite.
UPDATE db_schema
SET synced_at = (db.metadata ->> 'lastSyncTime')::timestamptz
FROM db
WHERE db.instance = db_schema.instance
  AND db.name = db_schema.db_name
  AND jsonb_typeof(db.metadata -> 'lastSyncTime') = 'string';
