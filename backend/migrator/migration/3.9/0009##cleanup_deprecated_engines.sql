-- Clean up deprecated engine references (OCEANBASE_ORACLE, RISINGWAVE, DM)
-- Mapping: RISINGWAVE -> POSTGRES, DM -> ORACLE, OCEANBASE_ORACLE -> ORACLE

-- Update instances table to handle any deprecated engine references
-- The engine field is stored in the metadata JSONB column, not as a direct column
UPDATE instance
SET metadata = jsonb_set(
    metadata,
    '{engine}',
    CASE 
        WHEN metadata->>'engine' = 'DM' THEN '"ORACLE"'::jsonb                     -- DM -> ORACLE
        WHEN metadata->>'engine' = 'RISINGWAVE' THEN '"POSTGRES"'::jsonb           -- RISINGWAVE -> POSTGRES (PostgreSQL-compatible)
        WHEN metadata->>'engine' = 'OCEANBASE_ORACLE' THEN '"ORACLE"'::jsonb       -- OCEANBASE_ORACLE -> ORACLE
        ELSE metadata->'engine'
    END
)
WHERE metadata->>'engine' IN ('DM', 'RISINGWAVE', 'OCEANBASE_ORACLE');

-- Update review_config table to remove deprecated engines from SQL review rules
UPDATE review_config 
SET payload = jsonb_set(
    payload::jsonb,
    '{sqlReviewRules}',
    COALESCE(
        (
            SELECT jsonb_agg(rule)
            FROM jsonb_array_elements(payload::jsonb->'sqlReviewRules') AS rule
            WHERE rule->>'engine' NOT IN ('OCEANBASE_ORACLE', 'RISINGWAVE', 'DM')
        ),
        '[]'::jsonb
    )
)
WHERE payload::jsonb->'sqlReviewRules' IS NOT NULL
  AND EXISTS (
    SELECT 1 
    FROM jsonb_array_elements(payload::jsonb->'sqlReviewRules') AS rule
    WHERE rule->>'engine' IN ('OCEANBASE_ORACLE', 'RISINGWAVE', 'DM')
  );

