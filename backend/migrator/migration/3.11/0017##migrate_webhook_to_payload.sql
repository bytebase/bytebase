-- Migrate project_webhook columns to payload.
-- This migration moves type, name, url, and event_list fields into the JSONB payload column.

-- Step 1: Backfill payload with existing column data
UPDATE project_webhook
SET payload = jsonb_build_object(
    'type', type,
    'title', name,
    'url', url,
    'activities', (
        SELECT jsonb_agg(event)
        FROM unnest(event_list) AS event
    ),
    'directMessage', COALESCE(payload->'directMessage', 'false'::jsonb)
)
WHERE payload = '{}' OR payload->'type' IS NULL;

-- Step 2: Drop the old columns
ALTER TABLE project_webhook DROP COLUMN IF EXISTS type;
ALTER TABLE project_webhook DROP COLUMN IF EXISTS name;
ALTER TABLE project_webhook DROP COLUMN IF EXISTS url;
ALTER TABLE project_webhook DROP COLUMN IF EXISTS event_list;
