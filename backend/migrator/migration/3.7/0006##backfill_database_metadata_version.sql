-- Backfill database metadata version field from revision table
UPDATE db
SET metadata = jsonb_set(
    COALESCE(metadata, '{}'::jsonb),
    '{version}',
    to_jsonb(COALESCE(
        (
            SELECT revision.version
            FROM revision
            WHERE revision.instance = db.instance 
                AND revision.db_name = db.name 
                AND deleted_at IS NOT NULL
            ORDER BY revision.version DESC
            LIMIT 1
        ),
        ''
    ))
)
WHERE EXISTS (
    SELECT 1
    FROM revision
    WHERE revision.instance = db.instance 
        AND revision.db_name = db.name 
        AND deleted_at IS NULL
);