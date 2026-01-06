-- Remove prev_sync_history_id column from changelog table
-- This column is no longer needed as we can use the previous changelog's sync_history_id
-- to get the "before" schema for comparison.

ALTER TABLE changelog DROP COLUMN IF EXISTS prev_sync_history_id;
