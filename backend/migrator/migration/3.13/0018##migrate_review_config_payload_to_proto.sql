-- Migrate SQL review rule payloads from flat JSON to typed proto format
-- Old format: payload was stored as a JSON string like "{\"maxLength\":64,\"format\":\"^[a-z]+$\"}"
-- New format: payload is a typed object like {"namingPayload":{"maxLength":64,"format":"^[a-z]+$"}}
-- Note: Old proto had 'string payload', so protojson stored it as a JSON string
-- This migration parses the JSON string and wraps it in the appropriate typed payload field

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT id, payload FROM review_config
    LOOP
        UPDATE review_config
        SET payload = jsonb_set(
            r.payload,
            '{sqlReviewRules}',
            (
                SELECT jsonb_agg(
                    CASE
                        -- Naming rules: parse JSON string and wrap in namingPayload
                        WHEN (rule->>'type') IN (
                            'NAMING_TABLE', 'NAMING_COLUMN', 'NAMING_INDEX_PK', 'NAMING_INDEX_UK',
                            'NAMING_INDEX_FK', 'NAMING_INDEX_IDX', 'NAMING_COLUMN_AUTO_INCREMENT',
                            'TABLE_DROP_NAMING_CONVENTION'
                        ) AND rule->>'payload' IS NOT NULL AND rule->>'payload' != '' AND rule->>'payload' != 'null' AND rule->>'payload' != '{}'
                        THEN jsonb_set(rule, '{namingPayload}', (rule->>'payload')::jsonb) - 'payload'

                        -- Number rules: parse JSON string and wrap in numberPayload
                        WHEN (rule->>'type') IN (
                            'STATEMENT_INSERT_ROW_LIMIT', 'STATEMENT_AFFECTED_ROW_LIMIT',
                            'STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT', 'STATEMENT_MAXIMUM_LIMIT_VALUE',
                            'STATEMENT_MAXIMUM_JOIN_TABLE_COUNT', 'STATEMENT_MAXIMUM_STATEMENTS_IN_TRANSACTION',
                            'COLUMN_MAXIMUM_CHARACTER_LENGTH', 'COLUMN_MAXIMUM_VARCHAR_LENGTH',
                            'COLUMN_AUTO_INCREMENT_INITIAL_VALUE',
                            'INDEX_KEY_NUMBER_LIMIT', 'INDEX_TOTAL_NUMBER_LIMIT',
                            'TABLE_TEXT_FIELDS_TOTAL_LENGTH', 'TABLE_LIMIT_SIZE', 'SYSTEM_COMMENT_LENGTH'
                        ) AND rule->>'payload' IS NOT NULL AND rule->>'payload' != '' AND rule->>'payload' != 'null' AND rule->>'payload' != '{}'
                        THEN jsonb_set(rule, '{numberPayload}', (rule->>'payload')::jsonb) - 'payload'

                        -- String array rules: parse JSON string and wrap in stringArrayPayload
                        WHEN (rule->>'type') IN (
                            'COLUMN_REQUIRED', 'COLUMN_TYPE_DISALLOW_LIST',
                            'INDEX_PRIMARY_KEY_TYPE_ALLOWLIST', 'INDEX_TYPE_ALLOW_LIST',
                            'SYSTEM_CHARSET_ALLOWLIST', 'SYSTEM_COLLATION_ALLOWLIST',
                            'SYSTEM_FUNCTION_DISALLOWED_LIST', 'TABLE_DISALLOW_DDL', 'TABLE_DISALLOW_DML'
                        ) AND rule->>'payload' IS NOT NULL AND rule->>'payload' != '' AND rule->>'payload' != 'null' AND rule->>'payload' != '{}'
                        THEN jsonb_set(rule, '{stringArrayPayload}', (rule->>'payload')::jsonb) - 'payload'

                        -- Comment convention rules: parse JSON string and wrap in commentConventionPayload
                        WHEN (rule->>'type') IN ('TABLE_COMMENT', 'COLUMN_COMMENT')
                        AND rule->>'payload' IS NOT NULL AND rule->>'payload' != '' AND rule->>'payload' != 'null' AND rule->>'payload' != '{}'
                        THEN jsonb_set(rule, '{commentConventionPayload}', (rule->>'payload')::jsonb) - 'payload'

                        -- Naming case rule: parse JSON string and wrap in namingCasePayload
                        WHEN (rule->>'type') = 'NAMING_IDENTIFIER_CASE'
                        AND rule->>'payload' IS NOT NULL AND rule->>'payload' != '' AND rule->>'payload' != 'null' AND rule->>'payload' != '{}'
                        THEN jsonb_set(rule, '{namingCasePayload}', (rule->>'payload')::jsonb) - 'payload'

                        -- String rule: parse JSON string, extract "level" field (or default to INDEX), and wrap in stringPayload
                        WHEN (rule->>'type') = 'STATEMENT_QUERY_MINIMUM_PLAN_LEVEL'
                        AND rule->>'payload' IS NOT NULL AND rule->>'payload' != '' AND rule->>'payload' != 'null' AND rule->>'payload' != '{}'
                        THEN jsonb_set(
                            rule,
                            '{stringPayload}',
                            jsonb_build_object('value', COALESCE((rule->>'payload')::jsonb->>'level', 'INDEX'))
                        ) - 'payload'

                        -- Rules with empty/null payload or no payload (like NAMING_FULLY_QUALIFIED,
                        -- STATEMENT_MAX_EXECUTION_TIME, COLUMN_CURRENT_TIME_COUNT_LIMIT) - just remove payload field
                        ELSE rule - 'payload'
                    END
                )
                FROM jsonb_array_elements(r.payload->'sqlReviewRules') AS rules(rule)
            )
        )
        WHERE id = r.id;
    END LOOP;
END $$;
