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
                            'TABLE_TEXT_FIELDS_TOTAL_LENGTH', 'TABLE_LIMIT_SIZE', 'SYSTEM_COMMENT_LENGTH',
                            'ADVICE_ONLINE_MIGRATION'
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

                        -- Rules with empty/null payload or no payload.
                        -- Full list:
                        -- TYPE_UNSPECIFIED
                        -- ENGINE_MYSQL_USE_INNODB
                        -- NAMING_FULLY_QUALIFIED
                        -- NAMING_TABLE_NO_KEYWORD
                        -- NAMING_IDENTIFIER_NO_KEYWORD
                        -- STATEMENT_SELECT_NO_SELECT_ALL
                        -- STATEMENT_WHERE_REQUIRE_SELECT
                        -- STATEMENT_WHERE_REQUIRE_UPDATE_DELETE
                        -- STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE
                        -- STATEMENT_DISALLOW_ON_DEL_CASCADE
                        -- STATEMENT_DISALLOW_RM_TBL_CASCADE
                        -- STATEMENT_DISALLOW_COMMIT
                        -- STATEMENT_DISALLOW_LIMIT
                        -- STATEMENT_DISALLOW_ORDER_BY
                        -- STATEMENT_MERGE_ALTER_TABLE
                        -- STATEMENT_INSERT_MUST_SPECIFY_COLUMN
                        -- STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND
                        -- STATEMENT_DML_DRY_RUN
                        -- STATEMENT_DISALLOW_ADD_COLUMN_WITH_DEFAULT
                        -- STATEMENT_ADD_CHECK_NOT_VALID
                        -- STATEMENT_ADD_FOREIGN_KEY_NOT_VALID
                        -- STATEMENT_DISALLOW_ADD_NOT_NULL
                        -- STATEMENT_SELECT_FULL_TABLE_SCAN
                        -- STATEMENT_CREATE_SPECIFY_SCHEMA
                        -- STATEMENT_CHECK_SET_ROLE_VARIABLE
                        -- STATEMENT_DISALLOW_USING_FILESORT
                        -- STATEMENT_DISALLOW_USING_TEMPORARY
                        -- STATEMENT_WHERE_NO_EQUAL_NULL
                        -- STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS
                        -- STATEMENT_JOIN_STRICT_COLUMN_ATTRS
                        -- STATEMENT_NON_TRANSACTIONAL
                        -- STATEMENT_ADD_COLUMN_WITHOUT_POSITION
                        -- STATEMENT_DISALLOW_OFFLINE_DDL
                        -- STATEMENT_DISALLOW_CROSS_DB_QUERIES
                        -- STATEMENT_MAX_EXECUTION_TIME
                        -- STATEMENT_REQUIRE_ALGORITHM_OPTION
                        -- STATEMENT_REQUIRE_LOCK_OPTION
                        -- STATEMENT_OBJECT_OWNER_CHECK
                        -- TABLE_REQUIRE_PK
                        -- TABLE_NO_FOREIGN_KEY
                        -- TABLE_DISALLOW_PARTITION
                        -- TABLE_DISALLOW_TRIGGER
                        -- TABLE_NO_DUPLICATE_INDEX
                        -- TABLE_DISALLOW_SET_CHARSET
                        -- TABLE_REQUIRE_CHARSET
                        -- TABLE_REQUIRE_COLLATION
                        -- COLUMN_NO_NULL
                        -- COLUMN_DISALLOW_CHANGE_TYPE
                        -- COLUMN_SET_DEFAULT_FOR_NOT_NULL
                        -- COLUMN_DISALLOW_CHANGE
                        -- COLUMN_DISALLOW_CHANGING_ORDER
                        -- COLUMN_DISALLOW_DROP
                        -- COLUMN_DISALLOW_DROP_IN_INDEX
                        -- COLUMN_AUTO_INCREMENT_MUST_INTEGER
                        -- COLUMN_DISALLOW_SET_CHARSET
                        -- COLUMN_AUTO_INCREMENT_MUST_UNSIGNED
                        -- COLUMN_CURRENT_TIME_COUNT_LIMIT
                        -- COLUMN_REQUIRE_DEFAULT
                        -- COLUMN_DEFAULT_DISALLOW_VOLATILE
                        -- COLUMN_ADD_NOT_NULL_REQUIRE_DEFAULT
                        -- COLUMN_REQUIRE_CHARSET
                        -- COLUMN_REQUIRE_COLLATION
                        -- SCHEMA_BACKWARD_COMPATIBILITY
                        -- DATABASE_DROP_EMPTY_DATABASE
                        -- INDEX_NO_DUPLICATE_COLUMN
                        -- INDEX_PK_TYPE_LIMIT
                        -- INDEX_TYPE_NO_BLOB
                        -- INDEX_CREATE_CONCURRENTLY
                        -- INDEX_NOT_REDUNDANT
                        -- SYSTEM_PROCEDURE_DISALLOW_CREATE
                        -- SYSTEM_EVENT_DISALLOW_CREATE
                        -- SYSTEM_VIEW_DISALLOW_CREATE
                        -- SYSTEM_FUNCTION_DISALLOW_CREATE
                        -- SYSTEM_FUNCTION_DISALLOWED_LIST
                        -- BUILTIN_PRIOR_BACKUP_CHECK
                        ELSE rule - 'payload'
                    END
                )
                FROM jsonb_array_elements(r.payload->'sqlReviewRules') AS rules(rule)
            )
        )
        WHERE id = r.id AND r.payload ->> 'sqlReviewRules' IS NOT NULL;
    END LOOP;
END $$;
