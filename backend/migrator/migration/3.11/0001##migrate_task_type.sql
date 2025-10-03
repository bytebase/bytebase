-- Migrate task types to consolidate DATABASE_SCHEMA_UPDATE, DATABASE_SCHEMA_UPDATE_GHOST,
-- DATABASE_DATA_UPDATE into DATABASE_MIGRATE with migrateType field
-- DATABASE_SCHEMA_UPDATE_SDL becomes DATABASE_SDL

-- Migrate DATABASE_SCHEMA_UPDATE to DATABASE_MIGRATE with migrateType=DDL
UPDATE task
SET payload = payload || jsonb_build_object('migrateType', 'DDL')
WHERE type = 'DATABASE_SCHEMA_UPDATE';

-- Migrate DATABASE_SCHEMA_UPDATE_GHOST to DATABASE_MIGRATE with migrateType=GHOST
UPDATE task
SET payload = payload || jsonb_build_object('migrateType', 'GHOST')
WHERE type = 'DATABASE_SCHEMA_UPDATE_GHOST';

-- Migrate DATABASE_DATA_UPDATE to DATABASE_MIGRATE with migrateType=DML
UPDATE task
SET payload = payload || jsonb_build_object('migrateType', 'DML')
WHERE type = 'DATABASE_DATA_UPDATE';

-- Update task type values
UPDATE task
SET type = 'DATABASE_MIGRATE'
WHERE type IN ('DATABASE_SCHEMA_UPDATE', 'DATABASE_SCHEMA_UPDATE_GHOST', 'DATABASE_DATA_UPDATE');

-- Rename DATABASE_SCHEMA_UPDATE_SDL to DATABASE_SDL
UPDATE task
SET type = 'DATABASE_SDL'
WHERE type = 'DATABASE_SCHEMA_UPDATE_SDL';
