ALTER TABLE project ADD COLUMN IF NOT EXISTS schema_change_type TEXT NOT NULL CHECK (schema_change_type IN ('DDL', 'SDL')) DEFAULT 'DDL';
