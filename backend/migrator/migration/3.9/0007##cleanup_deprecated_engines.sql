-- Clean up deprecated engine references (OCEANBASE_ORACLE, RISINGWAVE, DM)
-- Mapping: RISINGWAVE -> POSTGRES, DM -> ORACLE, OCEANBASE_ORACLE -> ORACLE

-- Update instances table to handle any deprecated engine references
UPDATE instance
SET engine = CASE 
    WHEN engine = 'DM' THEN 'ORACLE'                     -- DM -> ORACLE
    WHEN engine = 'RISINGWAVE' THEN 'POSTGRES'           -- RISINGWAVE -> POSTGRES (PostgreSQL-compatible)
    WHEN engine = 'OCEANBASE_ORACLE' THEN 'ORACLE'       -- OCEANBASE_ORACLE -> ORACLE
    ELSE engine
END
WHERE engine IN ('DM', 'RISINGWAVE', 'OCEANBASE_ORACLE');

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

