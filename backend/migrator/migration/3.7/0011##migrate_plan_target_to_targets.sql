-- Migrate plan config from single "target" to array "targets" for changeDatabaseConfig only
UPDATE plan
SET config = jsonb_set(
    config,
    '{specs}',
    (
        SELECT jsonb_agg(
            CASE
                -- Handle changeDatabaseConfig with target field
                WHEN spec_value->'changeDatabaseConfig' ? 'target' THEN
                    jsonb_set(
                        spec_value,
                        '{changeDatabaseConfig}',
                        jsonb_set(
                            (spec_value->'changeDatabaseConfig')::jsonb - 'target',
                            '{targets}',
                            jsonb_build_array(spec_value->'changeDatabaseConfig'->'target')
                        )
                    )
                -- Keep other specs unchanged
                ELSE spec_value
            END
        )
        FROM jsonb_array_elements(config->'specs') AS spec_value
    )
)
WHERE config ? 'specs'
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements(config->'specs') AS spec
    WHERE spec->'changeDatabaseConfig' ? 'target'
  );