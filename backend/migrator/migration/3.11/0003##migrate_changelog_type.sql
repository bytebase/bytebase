-- Migrate changelog type field to use new enum names
-- Old values: BASELINE, MIGRATE, MIGRATE_SDL, MIGRATE_GHOST, DATA
-- New values: BASELINE, DDL, DML, GHOST, SDL

-- Rename MIGRATE to DDL
UPDATE changelog
SET payload = jsonb_set(payload, '{type}', '"DDL"')
WHERE payload->>'type' = 'MIGRATE';

-- Rename MIGRATE_SDL to SDL
UPDATE changelog
SET payload = jsonb_set(payload, '{type}', '"SDL"')
WHERE payload->>'type' = 'MIGRATE_SDL';

-- Rename MIGRATE_GHOST to GHOST
UPDATE changelog
SET payload = jsonb_set(payload, '{type}', '"GHOST"')
WHERE payload->>'type' = 'MIGRATE_GHOST';

-- Rename DATA to DML
UPDATE changelog
SET payload = jsonb_set(payload, '{type}', '"DML"')
WHERE payload->>'type' = 'DATA';

-- BASELINE remains unchanged
