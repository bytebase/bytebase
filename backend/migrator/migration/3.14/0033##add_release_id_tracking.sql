-- Add columns for release ID tracking
ALTER TABLE release ADD COLUMN release_id TEXT NOT NULL DEFAULT '';
ALTER TABLE release ADD COLUMN train TEXT NOT NULL DEFAULT '';
ALTER TABLE release ADD COLUMN iteration INTEGER NOT NULL DEFAULT 0;

-- Backfill existing releases with meaningful values based on created_at
-- Group by (project, date) and assign iteration based on creation order
WITH numbered_releases AS (
  SELECT
    id,
    project,
    TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYYMMDD') AS date_str,
    ROW_NUMBER() OVER (
      PARTITION BY project, TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYYMMDD')
      ORDER BY created_at, id
    ) - 1 AS iteration_num
  FROM release
  WHERE release_id = ''
)
UPDATE release r
SET
  train = 'release_' || n.date_str || '-RC',
  iteration = n.iteration_num,
  release_id = 'release_' || n.date_str || '-RC' || LPAD(n.iteration_num::TEXT, 2, '0')
FROM numbered_releases n
WHERE r.id = n.id;

-- Create indexes after backfill to avoid unique constraint violations
CREATE UNIQUE INDEX idx_release_project_train_iteration ON release(project, train, iteration);
CREATE INDEX idx_release_project_release_id ON release(project, release_id);

-- Ensure columns are not nullable (already NOT NULL from ALTER TABLE, but this is explicit)
ALTER TABLE release ALTER COLUMN release_id SET NOT NULL;
ALTER TABLE release ALTER COLUMN train SET NOT NULL;
ALTER TABLE release ALTER COLUMN iteration SET NOT NULL;

-- Update plan.config JSONB to replace old release UIDs with new release_ids
-- Only update if changeDatabaseConfig.release exists and matches pattern
UPDATE plan p
SET config = jsonb_set(
  config,
  '{changeDatabaseConfig,release}',
  to_jsonb(CONCAT('projects/', r.project, '/releases/', r.release_id))
)
FROM release r
WHERE
  p.config->'changeDatabaseConfig'->>'release' IS NOT NULL
  AND p.config->'changeDatabaseConfig'->>'release' LIKE 'projects/%/releases/%'
  AND SPLIT_PART(p.config->'changeDatabaseConfig'->>'release', '/', 4)::bigint = r.id;

-- Update task.payload JSONB to replace old release UIDs with new release_ids
-- Only update if release field exists and matches pattern
UPDATE task t
SET payload = jsonb_set(
  t.payload,
  '{release}',
  to_jsonb(CONCAT('projects/', r.project, '/releases/', r.release_id))
)
FROM release r
WHERE
  t.payload->>'release' IS NOT NULL
  AND t.payload->>'release' LIKE 'projects/%/releases/%'
  AND SPLIT_PART(t.payload->>'release', '/', 4)::bigint = r.id;

-- Update revision.payload JSONB to replace old release UIDs with new release_ids
-- Only update if release field exists and matches pattern
UPDATE revision rv
SET payload = jsonb_set(
  rv.payload,
  '{release}',
  to_jsonb(CONCAT('projects/', r.project, '/releases/', r.release_id))
)
FROM release r
WHERE
  rv.payload->>'release' IS NOT NULL
  AND rv.payload->>'release' LIKE 'projects/%/releases/%'
  AND SPLIT_PART(rv.payload->>'release', '/', 4)::bigint = r.id;

-- Drop digest column from release table
ALTER TABLE release DROP COLUMN IF EXISTS digest;

-- Clean up title field from release.payload JSONB
UPDATE release
SET payload = payload - 'title'
WHERE payload ? 'title';
