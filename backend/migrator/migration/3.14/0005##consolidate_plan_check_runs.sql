-- Consolidate plan_check_run records: one record per plan instead of N×types

-- Step 1: Create temp table with deduplicated latest records (last 30 days only)
CREATE TEMP TABLE plan_check_run_deduped AS
SELECT DISTINCT ON (plan_id, type, config->>'instanceId', config->>'databaseName')
    *
FROM plan_check_run
WHERE created_at >= NOW() - INTERVAL '30 days'
  AND status != 'CANCELED'
ORDER BY plan_id, type, config->>'instanceId', config->>'databaseName', created_at DESC;

-- Step 2: Delete all old records
DELETE FROM plan_check_run;

-- Step 3: Insert consolidated records (one per plan)
INSERT INTO plan_check_run (plan_id, status, config, result, created_at, updated_at)
SELECT
    plan_id,
    -- Status aggregation
    CASE
        WHEN bool_or(status = 'RUNNING') THEN 'RUNNING'
        WHEN bool_or(status = 'FAILED') THEN 'FAILED'
        ELSE 'DONE'
    END,
    -- Config: if any RUNNING, empty (will be re-run); otherwise aggregate
    CASE
        WHEN bool_or(status = 'RUNNING') THEN '{"targets": []}'::jsonb
        ELSE jsonb_build_object('targets', (
            SELECT jsonb_agg(target)
            FROM (
                SELECT jsonb_build_object(
                    'instanceId', d2.config->>'instanceId',
                    'databaseName', d2.config->>'databaseName',
                    'sheetSha256', d2.config->>'sheetSha256',
                    'enablePriorBackup', COALESCE((d2.config->>'enablePriorBackup')::boolean, false),
                    'enableGhost', COALESCE((d2.config->>'enableGhost')::boolean, false),
                    'enableSdl', COALESCE((d2.config->>'enableSdl')::boolean, false),
                    'ghostFlags', COALESCE(d2.config->'ghostFlags', '{}'::jsonb),
                    'checkTypes', array_agg(
                        CASE d2.type
                            WHEN 'bb.plan-check.database.statement.advise' THEN 'PLAN_CHECK_TYPE_STATEMENT_ADVISE'
                            WHEN 'bb.plan-check.database.statement.summary.report' THEN 'PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT'
                            WHEN 'bb.plan-check.database.ghost.sync' THEN 'PLAN_CHECK_TYPE_GHOST_SYNC'
                        END
                    )
                ) as target
                FROM plan_check_run_deduped d2
                WHERE d2.plan_id = d.plan_id
                GROUP BY
                    d2.config->>'instanceId',
                    d2.config->>'databaseName',
                    d2.config->>'sheetSha256',
                    d2.config->>'enablePriorBackup',
                    d2.config->>'enableGhost',
                    d2.config->>'enableSdl',
                    d2.config->'ghostFlags'
            ) targets
        ))
    END,
    -- Results: empty if RUNNING, otherwise aggregate with type tagging
    CASE
        WHEN bool_or(status = 'RUNNING') THEN '{"results": []}'::jsonb
        ELSE jsonb_build_object('results', (
            SELECT COALESCE(jsonb_agg(
                r || jsonb_build_object(
                    'instanceId', d3.config->>'instanceId',
                    'databaseName', d3.config->>'databaseName',
                    'checkType', CASE d3.type
                        WHEN 'bb.plan-check.database.statement.advise' THEN 'PLAN_CHECK_TYPE_STATEMENT_ADVISE'
                        WHEN 'bb.plan-check.database.statement.summary.report' THEN 'PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT'
                        WHEN 'bb.plan-check.database.ghost.sync' THEN 'PLAN_CHECK_TYPE_GHOST_SYNC'
                    END
                )
            ), '[]'::jsonb)
            FROM plan_check_run_deduped d3
            LEFT JOIN LATERAL jsonb_array_elements(d3.result->'results') r ON true
            WHERE d3.plan_id = d.plan_id
        ))
    END,
    MAX(created_at),
    MAX(updated_at)
FROM plan_check_run_deduped d
GROUP BY plan_id;

-- Step 4: Cleanup temp table
DROP TABLE plan_check_run_deduped;

-- Step 5: Drop type column (no longer used)
ALTER TABLE plan_check_run DROP COLUMN IF EXISTS type;
