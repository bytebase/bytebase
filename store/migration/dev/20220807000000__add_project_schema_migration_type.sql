ALTER TABLE project ADD COLUMN schema_migration_type TEXT NOT NULL CHECK (schema_migration_type IN ('DDL', 'SDL')) DEFAULT 'DDL';
