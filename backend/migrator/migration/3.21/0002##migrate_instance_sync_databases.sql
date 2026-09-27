-- Migrate instance syncDatabases from a repeated field JSON shape:
--   "syncDatabases": ["db1", "db2"]
-- to the wrapper message JSON shape:
--   "syncDatabases": {"databases": ["db1", "db2"]}
--
-- The field remains absent when the old metadata did not store syncDatabases,
-- preserving the "sync all databases" default.
UPDATE instance
SET metadata = jsonb_set(
    metadata,
    '{syncDatabases}',
    jsonb_build_object('databases', metadata->'syncDatabases'),
    false
)
WHERE jsonb_typeof(metadata->'syncDatabases') = 'array';
