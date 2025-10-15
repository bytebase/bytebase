-- Migrate release files from changeType to migrationType
-- Also convert DDL_GHOST to GHOST
--
-- Old structure: payload.files[].changeType with values "DDL", "DDL_GHOST", "DML"
-- New structure: payload.files[].migrationType with values "DDL", "GHOST", "DML"

-- Simple string replacement: rename field and replace DDL_GHOST with GHOST
UPDATE release
SET payload = replace(
    replace(
        payload::text,
        '"changeType"',
        '"migrationType"'
    ),
    '"DDL_GHOST"',
    '"GHOST"'
)::jsonb
WHERE payload::text LIKE '%"changeType"%';
