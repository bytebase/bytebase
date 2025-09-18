-- Add labels column to project table for storing key-value labels
ALTER TABLE project ADD COLUMN IF NOT EXISTS labels JSONB DEFAULT '{}' NOT NULL;

-- Add GIN index for efficient label queries
CREATE INDEX IF NOT EXISTS idx_project_labels ON project USING GIN (labels);