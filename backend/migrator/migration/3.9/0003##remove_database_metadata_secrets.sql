-- Remove the 'secrets' field from DatabaseMetadata stored in db.metadata JSONB column
UPDATE db
SET metadata = metadata - 'secrets'
WHERE metadata ? 'secrets';