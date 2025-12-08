-- Migrate SQL review rule payloads from flat JSON to typed proto format
-- Old format: {"maxLength": 64, "format": "^[a-z]+$"}
-- New format: {"namingPayload": {"maxLength": 64, "format": "^[a-z]+$"}}

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
                        -- Naming rules: wrap payload in namingPayload
                        WHEN (rule->>'type') IN (
                            'NAMING_TABLE', 'NAMING_COLUMN', 'NAMING_INDEX_PK', 'NAMING_INDEX_UK',
                            'NAMING_INDEX_FK', 'NAMING_INDEX_IDX', 'NAMING_COLUMN_AUTO_INCREMENT',
                            'TABLE_DROP_NAMING_CONVENTION'
                        ) AND rule->'payload' IS NOT NULL AND rule->'payload' != 'null'::jsonb
                        THEN jsonb_set(rule, '{namingPayload}', rule->'payload') - 'payload'

                        -- Number rules: wrap payload in numberPayload
                        WHEN (rule->>'type') IN (
                            'STATEMENT_INSERT_ROW_LIMIT', 'STATEMENT_AFFECTED_ROW_LIMIT',
                            'STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT', 'STATEMENT_MAXIMUM_LIMIT_VALUE',
                            'STATEMENT_MAXIMUM_JOIN_TABLE_COUNT', 'STATEMENT_MAXIMUM_STATEMENTS_IN_TRANSACTION',
                            'STATEMENT_MAX_EXECUTION_TIME', 'STATEMENT_QUERY_MINIMUM_PLAN_LEVEL',
                            'COLUMN_MAXIMUM_CHARACTER_LENGTH', 'COLUMN_MAXIMUM_VARCHAR_LENGTH',
                            'COLUMN_AUTO_INCREMENT_INITIAL_VALUE', 'COLUMN_CURRENT_TIME_COUNT_LIMIT',
                            'INDEX_KEY_NUMBER_LIMIT', 'INDEX_TOTAL_NUMBER_LIMIT',
                            'TABLE_TEXT_FIELDS_TOTAL_LENGTH', 'TABLE_LIMIT_SIZE', 'SYSTEM_COMMENT_LENGTH'
                        ) AND rule->'payload' IS NOT NULL AND rule->'payload' != 'null'::jsonb
                        THEN jsonb_set(rule, '{numberPayload}', rule->'payload') - 'payload'

                        -- String array rules: wrap payload in stringArrayPayload
                        WHEN (rule->>'type') IN (
                            'COLUMN_REQUIRED', 'COLUMN_TYPE_DISALLOW_LIST',
                            'INDEX_PRIMARY_KEY_TYPE_ALLOWLIST', 'INDEX_TYPE_ALLOW_LIST',
                            'SYSTEM_CHARSET_ALLOWLIST', 'SYSTEM_COLLATION_ALLOWLIST',
                            'SYSTEM_FUNCTION_DISALLOWED_LIST', 'TABLE_DISALLOW_DDL', 'TABLE_DISALLOW_DML'
                        ) AND rule->'payload' IS NOT NULL AND rule->'payload' != 'null'::jsonb
                        THEN jsonb_set(rule, '{stringArrayPayload}', rule->'payload') - 'payload'

                        -- Comment convention rules: wrap payload in commentConventionPayload
                        WHEN (rule->>'type') IN ('TABLE_COMMENT', 'COLUMN_COMMENT')
                        AND rule->'payload' IS NOT NULL AND rule->'payload' != 'null'::jsonb
                        THEN jsonb_set(rule, '{commentConventionPayload}', rule->'payload') - 'payload'

                        -- Naming case rule: wrap payload in namingCasePayload
                        WHEN (rule->>'type') = 'NAMING_IDENTIFIER_CASE'
                        AND rule->'payload' IS NOT NULL AND rule->'payload' != 'null'::jsonb
                        THEN jsonb_set(rule, '{namingCasePayload}', rule->'payload') - 'payload'

                        -- String rule: wrap payload in stringPayload
                        WHEN (rule->>'type') = 'NAMING_FULLY_QUALIFIED'
                        AND rule->'payload' IS NOT NULL AND rule->'payload' != 'null'::jsonb
                        THEN jsonb_set(rule, '{stringPayload}', rule->'payload') - 'payload'

                        -- Rules with no payload or already migrated - leave as is
                        ELSE rule
                    END
                )
                FROM jsonb_array_elements(r.payload->'sqlReviewRules') AS rules(rule)
            )
        )
        WHERE id = r.id;
    END LOOP;
END $$;
