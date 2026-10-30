CREATE TABLE instance_schema (
    instance text PRIMARY KEY REFERENCES instance(resource_id),
    -- Stored as SchemaMetadata (github.com/bytebase/omni/metadata)
    metadata jsonb NOT NULL DEFAULT '{}'
);
