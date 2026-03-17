-- Add resource_id columns
ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS resource_id text;
UPDATE audit_log SET resource_id = gen_random_uuid()::text WHERE resource_id IS NULL;
ALTER TABLE audit_log ALTER COLUMN resource_id SET NOT NULL;
ALTER TABLE audit_log ALTER COLUMN resource_id SET DEFAULT gen_random_uuid()::text;
ALTER TABLE audit_log DROP CONSTRAINT IF EXISTS audit_log_pkey;
ALTER TABLE audit_log ADD PRIMARY KEY (resource_id);

ALTER TABLE query_history ADD COLUMN IF NOT EXISTS resource_id text;
UPDATE query_history SET resource_id = gen_random_uuid()::text WHERE resource_id IS NULL;
ALTER TABLE query_history ALTER COLUMN resource_id SET NOT NULL;
ALTER TABLE query_history ALTER COLUMN resource_id SET DEFAULT gen_random_uuid()::text;
ALTER TABLE query_history DROP CONSTRAINT IF EXISTS query_history_pkey;
ALTER TABLE query_history ADD PRIMARY KEY (resource_id);

ALTER TABLE export_archive ADD COLUMN IF NOT EXISTS resource_id text;
UPDATE export_archive SET resource_id = gen_random_uuid()::text WHERE resource_id IS NULL;
ALTER TABLE export_archive ALTER COLUMN resource_id SET NOT NULL;
ALTER TABLE export_archive ALTER COLUMN resource_id SET DEFAULT gen_random_uuid()::text;
ALTER TABLE export_archive DROP CONSTRAINT IF EXISTS export_archive_pkey;
ALTER TABLE export_archive ADD PRIMARY KEY (resource_id);

-- Backfill task_run.result JSONB: exportArchiveUid (int) → exportArchiveId (string uuid)
-- Only runs if export_archive.id column still exists (skipped on re-run after column drop)
DO $$ BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'export_archive' AND column_name = 'id'
  ) THEN
    EXECUTE '
      UPDATE task_run
      SET result = (result - ''exportArchiveUid'') || jsonb_build_object(''exportArchiveId'', ea.resource_id)
      FROM export_archive ea
      WHERE (result->>''exportArchiveUid'')::int = ea.id
        AND result ? ''exportArchiveUid''
    ';
  END IF;
END $$;

-- Drop the legacy id columns (no longer referenced)
ALTER TABLE audit_log DROP COLUMN IF EXISTS id;
ALTER TABLE query_history DROP COLUMN IF EXISTS id;
ALTER TABLE export_archive DROP COLUMN IF EXISTS id;
