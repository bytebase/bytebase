-- Split changelog type enum into Type and MigrationType
-- Old values: BASELINE, MIGRATE, MIGRATE_SDL, MIGRATE_GHOST, DATA
-- New structure:
--   Type enum: BASELINE, MIGRATE, SDL
--   MigrationType enum: DDL, DML, GHOST (only for MIGRATE type)

-- Update MIGRATE to MIGRATE with DDL migration_type
UPDATE changelog
SET payload = jsonb_set(
    jsonb_set(payload, '{type}', '"MIGRATE"'),
    '{migration_type}',
    '"DDL"'
)
WHERE payload->>'type' = 'MIGRATE';

-- Update MIGRATE_SDL to SDL (no migration_type needed)
UPDATE changelog
SET payload = jsonb_set(payload, '{type}', '"SDL"')
WHERE payload->>'type' = 'MIGRATE_SDL';

-- Update MIGRATE_GHOST to MIGRATE with GHOST migration_type
UPDATE changelog
SET payload = jsonb_set(
    jsonb_set(payload, '{type}', '"MIGRATE"'),
    '{migration_type}',
    '"GHOST"'
)
WHERE payload->>'type' = 'MIGRATE_GHOST';

-- Update DATA to MIGRATE with DML migration_type
UPDATE changelog
SET payload = jsonb_set(
    jsonb_set(payload, '{type}', '"MIGRATE"'),
    '{migration_type}',
    '"DML"'
)
WHERE payload->>'type' = 'DATA';

-- BASELINE remains unchanged
