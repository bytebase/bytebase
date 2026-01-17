-- Add category column to release table for filtering releases by category
ALTER TABLE release ADD COLUMN category TEXT NOT NULL DEFAULT '';

-- Create index for efficient category filtering
CREATE INDEX idx_release_category ON release(project_id, category) WHERE row_status = 'NORMAL';
