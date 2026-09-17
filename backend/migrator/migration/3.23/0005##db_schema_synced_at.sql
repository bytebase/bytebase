ALTER TABLE db_schema ADD COLUMN synced_at timestamptz NOT NULL DEFAULT to_timestamp(0);

-- A last sync time past the metadata database's own clock came from a replica
-- clock that ran ahead, and would hold off the metadata of every sync until the
-- database caught up with it. Drop it; the next sync records its own.
UPDATE db SET metadata = metadata - 'lastSyncTime'
WHERE (metadata->>'lastSyncTime')::timestamptz > now();
