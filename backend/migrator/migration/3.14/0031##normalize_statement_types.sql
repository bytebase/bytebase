-- Normalize legacy statement type strings to match proto enum names.
-- Changes:
--   "TRUNCATE_TABLE" -> "TRUNCATE" (Oracle/MSSQL legacy)
--
-- Background: Statement types were migrated from strings to proto enums.
-- Oracle and MSSQL parsers previously returned "TRUNCATE_TABLE" while
-- other parsers used "TRUNCATE". The new proto enum uses consistent naming.

UPDATE plan_check_run
SET result = replace(
    result::text,
    '"TRUNCATE_TABLE"',
    '"TRUNCATE"'
)::jsonb
WHERE result::text LIKE '%TRUNCATE_TABLE%';
