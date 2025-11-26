-- Migrate risk-based approval rules to direct approval rules
--
-- This migration converts the two-step approval evaluation (risk → approval rule)
-- into a single-step direct evaluation (approval rule with full CEL conditions).
--
-- Before:
--   Risk table: source + level + CEL expression (e.g., resource.environment_id == "prod")
--   Approval rules: source + level match → template (e.g., source == "DDL" && level == "HIGH")
--
-- After:
--   Approval rules: source enum + direct CEL condition → template
--   Risk table: No longer used for approval flow (kept for cleanup in later release)
--
-- The migration:
-- 1. For each risk (ordered by source, then level priority HIGH → MODERATE → LOW)
-- 2. Find the approval rule template that matches this risk's (source, level)
-- 3. Create a new approval rule with source enum + risk's CEL expression + template
-- 4. Add fallback rules (condition = "true") for UNSPECIFIED level per source

-- Helper: Convert risk source string to approval rule source enum name
CREATE OR REPLACE FUNCTION pg_temp.risk_source_to_enum(source_str text)
RETURNS text AS $$
BEGIN
    RETURN CASE source_str
        WHEN 'bb.risk.database.schema.update' THEN 'DDL'
        WHEN 'bb.risk.database.data.update' THEN 'DML'
        WHEN 'bb.risk.database.create' THEN 'CREATE_DATABASE'
        WHEN 'bb.risk.database.data.export' THEN 'EXPORT_DATA'
        WHEN 'bb.risk.request.role' THEN 'REQUEST_ROLE'
        ELSE 'SOURCE_UNSPECIFIED'
    END;
END;
$$ LANGUAGE plpgsql;

-- Helper: Convert approval rule source enum name to risk source string
CREATE OR REPLACE FUNCTION pg_temp.enum_to_risk_source(enum_str text)
RETURNS text AS $$
BEGIN
    RETURN CASE enum_str
        WHEN 'DDL' THEN 'bb.risk.database.schema.update'
        WHEN 'DML' THEN 'bb.risk.database.data.update'
        WHEN 'CREATE_DATABASE' THEN 'bb.risk.database.create'
        WHEN 'EXPORT_DATA' THEN 'bb.risk.database.data.export'
        WHEN 'REQUEST_ROLE' THEN 'bb.risk.request.role'
        ELSE ''
    END;
END;
$$ LANGUAGE plpgsql;

-- Helper: Get level priority (lower = higher priority)
CREATE OR REPLACE FUNCTION pg_temp.level_priority(level_str text)
RETURNS int AS $$
BEGIN
    RETURN CASE level_str
        WHEN 'HIGH' THEN 1
        WHEN 'MODERATE' THEN 2
        WHEN 'LOW' THEN 3
        WHEN 'RISK_LEVEL_UNSPECIFIED' THEN 4
        ELSE 5
    END;
END;
$$ LANGUAGE plpgsql;

-- Helper: Evaluate if a CEL expression matches given source and level
-- This is a simplified parser that handles common patterns like:
-- - source == "DDL" && level == "HIGH"
-- - (source == "DDL" && level == "HIGH") || (source == "DML" && level == "MODERATE")
CREATE OR REPLACE FUNCTION pg_temp.cel_matches_source_level(
    cel_expr text,
    source_val text,
    level_val text
)
RETURNS boolean AS $$
DECLARE
    clause text;
    source_in_clause boolean;
    level_in_clause boolean;
BEGIN
    -- Split expression by '||' (OR operator) and check each clause
    -- For a match, source and level must appear TOGETHER in the same AND clause
    FOR clause IN SELECT unnest(string_to_array(cel_expr, '||'))
    LOOP
        -- Check if this clause contains the source
        source_in_clause := (
            clause LIKE '%source == "' || source_val || '"%'
            OR clause LIKE '%source == ''' || source_val || '''%'
        );

        -- Check if this clause contains the level
        level_in_clause := (
            clause LIKE '%level == "' || level_val || '"%'
            OR clause LIKE '%level == ''' || level_val || '''%'
        );

        -- If both source and level are in the same clause, it's a match
        IF source_in_clause AND level_in_clause THEN
            RETURN true;
        END IF;
    END LOOP;

    RETURN false;
END;
$$ LANGUAGE plpgsql;

-- Main migration: Update WORKSPACE_APPROVAL setting
WITH
-- Step 1: Get all active risks with their source enum names
active_risks AS (
    SELECT
        r.id,
        r.name,
        r.source AS source_str,
        pg_temp.risk_source_to_enum(r.source) AS source_enum,
        r.level,
        r.expression->'expression' AS expression,
        pg_temp.level_priority(r.level) AS priority
    FROM risk r
    WHERE r.active = true
),

-- Step 2: Get the current approval setting
current_setting AS (
    SELECT
        value::jsonb AS setting_json
    FROM setting
    WHERE name = 'WORKSPACE_APPROVAL'
),

-- Step 3: Extract current approval rules (old format)
old_rules AS (
    SELECT
        rule_idx,
        rule->'condition'->>'expression' AS condition_expr,
        rule->'template' AS template
    FROM current_setting,
         jsonb_array_elements(setting_json->'rules') WITH ORDINALITY AS arr(rule, rule_idx)
),

-- Step 4: Match risks to approval rule templates
-- For each risk, find the approval rule template that matches its (source, level)
risk_to_template AS (
    SELECT DISTINCT ON (ar.id)
        ar.id AS risk_id,
        ar.source_enum,
        ar.expression,
        ar.priority,
        jsonb_set(orr.template, '{title}', to_jsonb(ar.name::TEXT)) AS template
    FROM active_risks ar
    CROSS JOIN old_rules orr
    WHERE pg_temp.cel_matches_source_level(orr.condition_expr, ar.source_enum, ar.level)
    ORDER BY ar.id, orr.rule_idx  -- Take the first matching rule (preserve rule ordering)
),

-- Step 5: Get distinct sources that have approval rules
sources_with_rules AS (
    SELECT DISTINCT source_enum FROM risk_to_template
),

-- Step 6: Add virtual UNSPECIFIED fallback rules
-- These match when no specific risk matches (condition = "true")
unspecified_fallbacks AS (
    SELECT
        swr.source_enum,
        '"true"'::jsonb AS expression,
        100 AS priority,  -- Lower priority than any actual risk
        jsonb_set(orr.template, '{title}', to_jsonb('Fallback rule'::TEXT)) AS template
    FROM sources_with_rules swr
    CROSS JOIN old_rules orr
    WHERE pg_temp.cel_matches_source_level(orr.condition_expr, swr.source_enum, 'RISK_LEVEL_UNSPECIFIED')
),

-- Step 7: Combine actual risks and fallbacks, ordered by source then priority
all_new_rules AS (
    -- Actual risks
    SELECT
        source_enum,
        expression,
        priority,
        template
    FROM risk_to_template

    UNION ALL

    -- UNSPECIFIED fallbacks
    SELECT
        source_enum,
        expression,
        priority,
        template
    FROM unspecified_fallbacks

    ORDER BY source_enum, priority
),

-- Step 8: Build the new rules array with explicit ordering
new_rules_json AS (
    SELECT jsonb_agg(
        jsonb_build_object(
            'source', source_enum,
            'condition', jsonb_build_object('expression', expression),
            'template', template
        )
        ORDER BY source_enum, priority
    ) AS rules
    FROM all_new_rules
)

-- Step 9: Update the setting with new rules
UPDATE setting
SET value = jsonb_build_object('rules', COALESCE(nr.rules, '[]'::jsonb))::text
FROM new_rules_json nr
WHERE name = 'WORKSPACE_APPROVAL'
  AND EXISTS (SELECT 1 FROM current_setting);

-- Handle case where WORKSPACE_APPROVAL setting doesn't exist or has no rules
-- (no update needed, approval flow will work without rules - no approval required)
