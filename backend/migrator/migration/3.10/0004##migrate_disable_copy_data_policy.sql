-- Migrate DISABLE_COPY_DATA policies to QUERY_DATA policies with disable_copy_data field
-- This migration converts the deprecated DISABLE_COPY_DATA policy type to QUERY_DATA

-- First, handle any existing QUERY_DATA policies for resources that also have DISABLE_COPY_DATA
-- We need to merge them by adding the disable_copy_data field
UPDATE policy p1
SET
    payload = jsonb_set(
        COALESCE(p1.payload, '{}'::jsonb),
        '{disableCopyData}',
        COALESCE((
            SELECT p2.payload->'active'
            FROM policy p2
            WHERE p2.resource = p1.resource
              AND p2.type = 'DISABLE_COPY_DATA'
            LIMIT 1
        ), 'false'::jsonb)
    ),
    updated_at = NOW()
WHERE p1.type = 'QUERY_DATA'
  AND EXISTS (
    SELECT 1
    FROM policy p2
    WHERE p2.resource = p1.resource
      AND p2.type = 'DISABLE_COPY_DATA'
  );

-- Delete the now-redundant DISABLE_COPY_DATA policies that were merged
DELETE FROM policy
WHERE type = 'DISABLE_COPY_DATA'
  AND resource IN (
    SELECT resource
    FROM policy
    WHERE type = 'QUERY_DATA'
  );

-- Convert remaining DISABLE_COPY_DATA policies to QUERY_DATA
UPDATE policy
SET
    type = 'QUERY_DATA',
    payload = jsonb_build_object(
        'disableCopyData',
        COALESCE(payload->'active', 'false'::jsonb)
    ),
    updated_at = NOW()
WHERE type = 'DISABLE_COPY_DATA';