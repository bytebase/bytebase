-- Migration: Convert classification level from string to int.
--
-- 1. Build a temp mapping from old string level IDs to new integer levels (array index + 1).
-- 2. Migrate DATA_CLASSIFICATION setting: remove id from levels, add level (int).
--    Convert classification levelId (string) to level (int).
-- 3. Migrate MASKING_RULE policy CEL expressions: replace string level IDs with integers
--    using the mapping from step 1.

-- Step 1: Build level_id -> level_number mapping from existing classification configs.
-- Each workspace may have its own classification config with its own level IDs.
CREATE TEMPORARY TABLE _level_id_map (
    workspace TEXT,
    old_level_id TEXT,
    new_level INT
);

INSERT INTO _level_id_map (workspace, old_level_id, new_level)
SELECT
    s.workspace,
    lv.val->>'id' AS old_level_id,
    lv.pos::int AS new_level
FROM setting s,
     jsonb_array_elements(s.value->'configs') AS config,
     LATERAL (
         SELECT e.value AS val, e.ordinality AS pos
         FROM jsonb_array_elements(config->'levels') WITH ORDINALITY AS e(value, ordinality)
     ) lv
WHERE s.name = 'DATA_CLASSIFICATION'
  AND s.value->'configs' IS NOT NULL;

-- Step 2: Migrate DATA_CLASSIFICATION setting.
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

-- Step 3: Migrate MASKING_RULE policy CEL expressions using the level_id -> int mapping.
-- Uses a PL/pgSQL block to iterate over each masking rule and replace each old string
-- level ID with its numeric equivalent. This handles arbitrary string IDs (e.g. "S2")
-- and only touches classification_level operands, leaving other parts of the expression intact.
DO $$
DECLARE
    pol RECORD;
    rule_val JSONB;
    expr_text TEXT;
    new_expr TEXT;
    mapping RECORD;
    new_rules JSONB;
    rules_changed BOOLEAN;
    escaped_id TEXT;
    bracket_content TEXT;
    new_bracket TEXT;
    matches TEXT[];
BEGIN
    FOR pol IN
        SELECT ctid, payload, workspace
        FROM policy
        WHERE type = 'MASKING_RULE'
          AND payload->'rules' IS NOT NULL
          AND payload::text LIKE '%classification_level%'
    LOOP
        new_rules := '[]'::jsonb;
        rules_changed := false;

        FOR rule_val IN SELECT jsonb_array_elements(pol.payload->'rules')
        LOOP
            expr_text := rule_val->'condition'->>'expression';

            IF expr_text IS NOT NULL AND expr_text ~ 'classification_level' THEN
                new_expr := expr_text;

                FOR mapping IN
                    SELECT old_level_id, new_level
                    FROM _level_id_map
                    WHERE workspace = pol.workspace
                    ORDER BY length(old_level_id) DESC
                LOOP
                    -- Escape regex special characters in the old level ID.
                    escaped_id := regexp_replace(mapping.old_level_id, '([\.\+\*\?\[\]\(\)\{\}\|\\^$])', '\\\1', 'g');

                    -- Replace classification_level == "old_id" and != "old_id".
                    new_expr := regexp_replace(
                        new_expr,
                        'classification_level(\s*(?:==|!=)\s*)"' || escaped_id || '"',
                        'classification_level\1' || mapping.new_level::text,
                        'g'
                    );
                END LOOP;

                -- Replace quoted level IDs inside classification_level in [...] brackets.
                -- Extract the bracket content, do replacements only within it, then reassemble.
                matches := regexp_match(new_expr, '(classification_level\s+in\s+\[)([^\]]+)(\])');
                IF matches IS NOT NULL THEN
                    bracket_content := matches[2];
                    new_bracket := bracket_content;
                    FOR mapping IN
                        SELECT old_level_id, new_level
                        FROM _level_id_map
                        WHERE workspace = pol.workspace
                        ORDER BY length(old_level_id) DESC
                    LOOP
                        new_bracket := replace(new_bracket, '"' || mapping.old_level_id || '"', mapping.new_level::text);
                    END LOOP;
                    new_expr := replace(new_expr, matches[1] || bracket_content || matches[3], matches[1] || new_bracket || matches[3]);
                END IF;

                IF new_expr IS DISTINCT FROM expr_text THEN
                    rule_val := jsonb_set(rule_val, '{condition,expression}', to_jsonb(new_expr));
                    rules_changed := true;
                END IF;
            END IF;

            new_rules := new_rules || jsonb_build_array(rule_val);
        END LOOP;

        IF rules_changed THEN
            UPDATE policy
            SET payload = jsonb_set(pol.payload, '{rules}', new_rules)
            WHERE ctid = pol.ctid;
        END IF;
    END LOOP;
END $$;

DROP TABLE _level_id_map;
