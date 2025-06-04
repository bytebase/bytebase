-- Drop the existing check constraint for task type
ALTER TABLE task DROP CONSTRAINT IF EXISTS task_type_check;

-- Backfill task type from string values to enum values
UPDATE task
SET type = CASE
    WHEN type = 'bb.task.database.create' THEN 'DATABASE_CREATE'
    WHEN type = 'bb.task.database.schema.update' THEN 'DATABASE_SCHEMA_UPDATE'
    WHEN type = 'bb.task.database.schema.update-ghost' THEN 'DATABASE_SCHEMA_UPDATE_GHOST'
    WHEN type = 'bb.task.database.data.update' THEN 'DATABASE_DATA_UPDATE'
    WHEN type = 'bb.task.database.data.export' THEN 'DATABASE_EXPORT'
END
WHERE type IS NOT NULL;