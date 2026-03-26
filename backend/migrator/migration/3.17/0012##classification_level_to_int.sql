-- Migration: Convert classification level from string to int.
--
-- 1. In DATA_CLASSIFICATION setting: remove id from levels, add level (int) based on
--    array position. Convert classification levelId (string) to level (int).
-- 2. In MASKING_RULE policies: convert string classification_level values in CEL
--    expressions to integers.

-- Step 1: Migrate DATA_CLASSIFICATION setting.
UPDATE setting
SET value = (
    SELECT jsonb_set(
        value,
        '{configs}',
        (
            SELECT COALESCE(jsonb_agg(
                jsonb_set(
                    jsonb_set(
                        config,
                        '{levels}',
                        (
                            SELECT COALESCE(jsonb_agg(
                                jsonb_build_object(
                                    'title', lv.val->>'title',
                                    'description', COALESCE(lv.val->>'description', ''),
                                    'level', lv.pos::int
                                )
                            ORDER BY lv.pos), '[]'::jsonb)
                            FROM (
                                SELECT e.value AS val, e.ordinality AS pos
                                FROM jsonb_array_elements(config->'levels') WITH ORDINALITY AS e(value, ordinality)
                            ) lv
                        )
                    ),
                    '{classification}',
                    (
                        SELECT COALESCE(jsonb_object_agg(
                            key,
                            CASE
                                WHEN cls_value ? 'levelId' THEN
                                    (cls_value - 'levelId') || jsonb_build_object(
                                        'level',
                                        (
                                            SELECT lr.pos::int
                                            FROM (
                                                SELECT e.value->>'id' AS level_id, e.ordinality AS pos
                                                FROM jsonb_array_elements(config->'levels') WITH ORDINALITY AS e(value, ordinality)
                                            ) lr
                                            WHERE lr.level_id = cls_value->>'levelId'
                                        )
                                    )
                                ELSE cls_value
                            END
                        ), '{}'::jsonb)
                        FROM jsonb_each(config->'classification') AS cls(key, cls_value)
                    )
                )
            ), '[]'::jsonb)
            FROM jsonb_array_elements(value->'configs') AS config
        )
    )
)
WHERE name = 'DATA_CLASSIFICATION'
  AND value->'configs' IS NOT NULL;

-- Step 2: Migrate MASKING_RULE policy CEL expressions.
-- Convert classification_level == "N" and != "N" to use integers.
UPDATE policy
SET payload = (
    SELECT jsonb_set(
        payload,
        '{rules}',
        (
            SELECT COALESCE(jsonb_agg(
                CASE
                    WHEN rule->'condition'->>'expression' ~ 'classification_level'
                    THEN jsonb_set(
                        rule,
                        '{condition,expression}',
                        to_jsonb(
                            regexp_replace(
                                rule->'condition'->>'expression',
                                'classification_level\s*(==|!=)\s*"(\d+)"',
                                'classification_level \1 \2',
                                'g'
                            )
                        )
                    )
                    ELSE rule
                END
            ), '[]'::jsonb)
            FROM jsonb_array_elements(payload->'rules') AS rule
        )
    )
)
WHERE type = 'MASKING_RULE'
  AND payload->'rules' IS NOT NULL
  AND payload::text LIKE '%classification_level%';

-- Step 2b: Strip quoted digits in classification_level in [...] expressions.
UPDATE policy
SET payload = (
    SELECT jsonb_set(
        payload,
        '{rules}',
        (
            SELECT COALESCE(jsonb_agg(
                CASE
                    WHEN rule->'condition'->>'expression' ~ 'classification_level\s+in'
                    THEN jsonb_set(
                        rule,
                        '{condition,expression}',
                        to_jsonb(
                            regexp_replace(
                                rule->'condition'->>'expression',
                                '"(\d+)"',
                                '\1',
                                'g'
                            )
                        )
                    )
                    ELSE rule
                END
            ), '[]'::jsonb)
            FROM jsonb_array_elements(payload->'rules') AS rule
        )
    )
)
WHERE type = 'MASKING_RULE'
  AND payload->'rules' IS NOT NULL
  AND payload::text LIKE '%classification_level%';
