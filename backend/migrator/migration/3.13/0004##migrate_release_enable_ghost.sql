-- Migrate release payloads: set enableGhost=true in files where migrationType="GHOST"
UPDATE release
SET payload = (
    SELECT jsonb_set(
        payload,
        '{files}',
        (
            SELECT jsonb_agg(
                CASE
                    WHEN file->>'migrationType' = 'GHOST' THEN
                        file || '{"enableGhost": true}'
                    ELSE file
                END
            )
            FROM jsonb_array_elements(payload->'files') AS file
        )
    )
)
WHERE payload->'files' IS NOT NULL
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements(payload->'files') AS file
    WHERE file->>'migrationType' = 'GHOST'
);
