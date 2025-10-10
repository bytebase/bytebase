-- Migrate risk level from numeric values (100, 200, 300) to string enum values
--
-- This migration converts the risk.level column from using numeric values
-- to proper string enum values that match the proto definition:
--   'RISK_LEVEL_UNSPECIFIED' = RISK_LEVEL_UNSPECIFIED (was 0)
--   'LOW' = LOW (was 100)
--   'MODERATE' = MODERATE (was 200)
--   'HIGH' = HIGH (was 300)
--
-- The migration updates:
-- 1. risk table: Convert level column from bigint to text with string values
-- 2. issue payload: Convert risk levels in approval flows
-- 3. setting table: Convert risk levels in workspace approval settings

-- Step 1: Add a temporary text column
ALTER TABLE risk ADD COLUMN level_text text;

-- Step 2: Convert numeric values to string values
UPDATE risk
SET level_text = CASE
    WHEN level = 0 THEN 'RISK_LEVEL_UNSPECIFIED'
    WHEN level = 100 THEN 'LOW'
    WHEN level = 200 THEN 'MODERATE'
    WHEN level = 300 THEN 'HIGH'
    ELSE 'RISK_LEVEL_UNSPECIFIED'
END;

-- Step 3: Drop the old numeric column
ALTER TABLE risk DROP COLUMN level;

-- Step 4: Rename the text column to level
ALTER TABLE risk RENAME COLUMN level_text TO level;

-- Step 5: Add NOT NULL constraint
ALTER TABLE risk ALTER COLUMN level SET NOT NULL;

-- Update workspace approval settings
-- The setting.value contains rules[].condition with level comparisons
-- Since these are CEL expressions stored as strings, we need to replace:
-- 'level == 100' -> 'level == "LOW"'
-- 'level == 200' -> 'level == "MODERATE"'
-- 'level == 300' -> 'level == "HIGH"'
-- 'level == 0' -> 'level == "RISK_LEVEL_UNSPECIFIED"'
UPDATE setting
SET value = (
    SELECT jsonb_build_object(
        'rules',
        (
            SELECT jsonb_agg(
                jsonb_build_object(
                    'condition', (
                        CASE
                            -- If expression exists, build condition with only the expression field
                            WHEN rule->'condition'->>'expression' IS NOT NULL THEN
                                jsonb_build_object(
                                    'expression',
                                    replace(
                                        replace(
                                            replace(
                                                replace(rule->'condition'->>'expression', 'level == 100', 'level == "LOW"'),
                                                'level == 200', 'level == "MODERATE"'
                                            ),
                                            'level == 300', 'level == "HIGH"'
                                        ),
                                        'level == 0', 'level == "RISK_LEVEL_UNSPECIFIED"'
                                    )
                                )
                            -- Otherwise, keep the original condition (likely empty object {})
                            ELSE COALESCE(rule->'condition', '{}'::jsonb)
                        END
                    ),
                    'template', rule->'template'
                )
            )
            FROM jsonb_array_elements((value::jsonb)->'rules') AS rule
        )
    )::text
)
WHERE name = 'WORKSPACE_APPROVAL'
  AND value::jsonb->'rules' IS NOT NULL
  AND value::text LIKE '%level == %';

