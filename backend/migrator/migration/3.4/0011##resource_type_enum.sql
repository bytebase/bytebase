ALTER TABLE policy ALTER COLUMN resource_type TYPE text;
DROP TYPE resource_type;
ALTER TABLE policy ADD CONSTRAINT resource_type_type CHECK (resource_type IN ('WORKSPACE', 'ENVIRONMENT', 'PROJECT', 'INSTANCE'));