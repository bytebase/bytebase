-- Add payload column to worksheet_organizer table and migrate starred column into it
--
-- This migration adds a "payload" column to store additional data for worksheets,
-- including the starred status and folders. The starred column is migrated into the payload.
ALTER TABLE worksheet_organizer DROP COLUMN IF EXISTS category;

-- Add payload column with default empty object
ALTER TABLE worksheet_organizer ADD COLUMN payload jsonb NOT NULL DEFAULT '{}';

-- Migrate existing starred values into the payload
UPDATE worksheet_organizer SET payload = jsonb_set(payload, '{starred}', to_jsonb(starred));

-- Drop the starred column as it's now in the payload
ALTER TABLE worksheet_organizer DROP COLUMN starred;

-- Create GIN index for efficient querying by principal_id and folders
CREATE INDEX idx_worksheet_organizer_principal_id_folders ON worksheet_organizer USING GIN(principal_id, (payload->'folders'));
