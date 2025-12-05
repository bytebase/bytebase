-- Change setting.value column from text to jsonb
-- All setting values are already JSON strings serialized by protojson.Marshal
ALTER TABLE setting ALTER COLUMN value TYPE jsonb USING value::jsonb;
