-- Migrate SQL review rule payloads from JSON string to typed proto format
-- This is a no-op migration for the database schema itself, as the column
-- remains JSONB. The actual data conversion happens at the application layer
-- when rules are loaded and saved.

-- The migration strategy is:
-- 1. Application code now writes typed proto payloads (via ConvertJSONPayloadToProto)
-- 2. Application code reads both old JSON strings and new typed protos (via GetXXXPayload)
-- 3. This migration is a marker for the schema version
-- 4. Future migration (after all instances upgraded) can clean up old format

-- No actual SQL changes needed - the JSONB column stores both formats.
-- Proto payloads are stored as camelCased JSON (e.g., "namingPayload": {...})
-- Old format is stored as JSON objects (e.g., {"maxLength": 64, "format": "..."})

-- Mark migration as applied
SELECT 1;
