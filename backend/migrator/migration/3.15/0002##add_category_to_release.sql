-- Add category column to release table for filtering releases by category
ALTER TABLE release ADD COLUMN category TEXT NOT NULL DEFAULT '';

-- Backfill existing releases with category = 'release'
UPDATE release SET category = 'release' WHERE category = '';

-- Create index for efficient category filtering
CREATE INDEX idx_release_category ON release(project, category);
