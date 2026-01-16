-- Backfill db.metadata.release from latest revision for databases with existing revisions
UPDATE db
SET metadata = jsonb_set(
  COALESCE(metadata, '{}'::jsonb),
  '{release}',
  to_jsonb((
    SELECT r.payload->>'release'
    FROM revision r
    WHERE r.instance = db.instance
      AND r.db_name = db.name
      AND r.deleted_at IS NULL
      AND r.payload->>'release' IS NOT NULL
    ORDER BY r.created_at DESC
    LIMIT 1
  ))
)
WHERE EXISTS (
  SELECT 1 FROM revision r
  WHERE r.instance = db.instance
    AND r.db_name = db.name
    AND r.deleted_at IS NULL
    AND r.payload->>'release' IS NOT NULL
);

-- Remove deprecated version field
UPDATE db SET metadata = metadata - 'version' WHERE metadata ? 'version';
