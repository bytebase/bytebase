-- Strip the resource name prefix from revision.payload.file, leaving only the plain file path.
-- Old format: projects/{project}/releases/{release}/files/{path}
-- New format: {path}
-- The regexp_replace extracts everything after "files/" as the plain path.
-- Note: Old data was never URL-encoded, so no decoding is needed.
UPDATE revision
SET payload = jsonb_set(
    payload,
    '{file}',
    to_jsonb(
        regexp_replace(
            payload->>'file',
            '^projects/[^/]+/releases/[^/]+/files/',
            ''
        )
    )
)
WHERE payload->>'file' LIKE 'projects/%/releases/%/files/%';
