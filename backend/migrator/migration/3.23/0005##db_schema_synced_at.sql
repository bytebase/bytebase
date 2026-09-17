ALTER TABLE db_schema ADD COLUMN synced_at timestamptz NOT NULL DEFAULT to_timestamp(0);

-- A last sync time past the metadata database's own clock came from a replica
-- clock that ran ahead, and would hold off the metadata of every sync until the
-- database caught up with it. Drop it; the next sync records its own.
UPDATE db SET metadata = metadata - 'lastSyncTime'
WHERE (metadata->>'lastSyncTime')::timestamptz > now();

-- The schema fence is this sequence, not synced_at itself: a wall clock can run
-- backward across a failover or a step correction, and clock_timestamp() would
-- then silently lose every sync of that database until it caught up. A
-- sequence only ever grows, on whichever replica reads it.
CREATE SEQUENCE db_schema_sync_seq;
ALTER TABLE db_schema ADD COLUMN sync_token bigint NOT NULL DEFAULT 0;
