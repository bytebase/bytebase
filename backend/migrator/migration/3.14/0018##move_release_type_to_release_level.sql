-- Move SchemaChangeType from per-file to release level in JSONB payloads.
-- Extract type from first file and set it at release level, then remove from all files.

UPDATE release
SET payload = jsonb_set(
    payload #- '{files}',
    '{type}',
    COALESCE(
        payload #> '{files,0,type}',
        '0'::jsonb  -- Default to SCHEMA_CHANGE_TYPE_UNSPECIFIED if no files
    )
) || jsonb_build_object(
    'files',
    (
        SELECT jsonb_agg(file_obj - 'type')
        FROM jsonb_array_elements(payload -> 'files') AS file_obj
    )
)
WHERE payload -> 'files' IS NOT NULL
  AND jsonb_array_length(payload -> 'files') > 0;
