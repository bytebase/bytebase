-- Add category column to worksheet_organizer table
--
-- This migration adds a "category" column to allow users to categorize their worksheets.
-- The column is NOT NULL with an empty string as the default value.

ALTER TABLE worksheet_organizer ADD COLUMN category text NOT NULL DEFAULT '';
