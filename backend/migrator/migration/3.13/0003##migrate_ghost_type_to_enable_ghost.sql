-- Migrate task payloads: convert migrateType="GHOST" to enableGhost=true
UPDATE task
SET payload = payload || '{"enableGhost": true}'
WHERE payload->>'migrateType' = 'GHOST';

-- Migrate plan config: set enableGhost=true in changeDatabaseConfig where migrateType="GHOST"
UPDATE plan
SET config = (
    SELECT jsonb_set(
        config,
        '{steps}',
        (
            SELECT jsonb_agg(
                CASE
                    WHEN step->'specs' IS NOT NULL THEN
                        jsonb_set(
                            step,
                            '{specs}',
                            (
                                SELECT jsonb_agg(
                                    CASE
                                        WHEN spec->'changeDatabaseConfig'->>'migrateType' = 'GHOST' THEN
                                            jsonb_set(
                                                spec,
                                                '{changeDatabaseConfig,enableGhost}',
                                                'true'::jsonb
                                            )
                                        ELSE spec
                                    END
                                )
                                FROM jsonb_array_elements(step->'specs') AS spec
                            )
                        )
                    ELSE step
                END
            )
            FROM jsonb_array_elements(config->'steps') AS step
        )
    )
)
WHERE config->'steps' IS NOT NULL
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements(config->'steps') AS step,
         jsonb_array_elements(step->'specs') AS spec
    WHERE spec->'changeDatabaseConfig'->>'migrateType' = 'GHOST'
);
