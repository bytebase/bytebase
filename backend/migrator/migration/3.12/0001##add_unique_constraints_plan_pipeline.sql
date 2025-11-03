-- Clean up duplicate issues with same plan_id (keep the oldest issue, delete newer ones)
-- This handles the case where multiple issues reference the same plan
DO $$
DECLARE
    dup_plan_id BIGINT;
    keep_issue_id INT;
    delete_issue_ids INT[];
BEGIN
    -- For each plan_id that has multiple issues
    FOR dup_plan_id IN
        SELECT plan_id
        FROM issue
        WHERE plan_id IS NOT NULL
        GROUP BY plan_id
        HAVING COUNT(*) > 1
    LOOP
        -- Find the oldest issue (to keep) and others (to delete)
        SELECT MIN(id) INTO keep_issue_id
        FROM issue
        WHERE plan_id = dup_plan_id;

        SELECT ARRAY_AGG(id) INTO delete_issue_ids
        FROM issue
        WHERE plan_id = dup_plan_id AND id != keep_issue_id;

        -- Log what we're doing
        RAISE NOTICE 'Duplicate issues for plan_id %: keeping issue % and deleting issues %',
            dup_plan_id, keep_issue_id, delete_issue_ids;

        -- Delete issue comments for duplicate issues
        DELETE FROM issue_comment WHERE issue_id = ANY(delete_issue_ids);

        -- Delete duplicate issues
        DELETE FROM issue WHERE id = ANY(delete_issue_ids);
    END LOOP;
END $$;

-- Clean up duplicate plans with same pipeline_id (keep the oldest plan, reassign references)
DO $$
DECLARE
    dup_pipeline_id INT;
    keep_plan_id BIGINT;
    delete_plan_ids BIGINT[];
    del_plan_id BIGINT;
BEGIN
    -- For each pipeline_id that has multiple plans
    FOR dup_pipeline_id IN
        SELECT pipeline_id
        FROM plan
        WHERE pipeline_id IS NOT NULL
        GROUP BY pipeline_id
        HAVING COUNT(*) > 1
    LOOP
        -- Find the oldest plan (to keep) and others (to delete)
        SELECT MIN(id) INTO keep_plan_id
        FROM plan
        WHERE pipeline_id = dup_pipeline_id;

        SELECT ARRAY_AGG(id) INTO delete_plan_ids
        FROM plan
        WHERE pipeline_id = dup_pipeline_id AND id != keep_plan_id;

        -- Log what we're doing
        RAISE NOTICE 'Duplicate plans for pipeline_id %: keeping plan % and deleting plans %',
            dup_pipeline_id, keep_plan_id, delete_plan_ids;

        -- Delete issues that reference the duplicate plans
        -- We can't reassign them because it would create duplicate issues for the same plan_id
        DELETE FROM issue_comment
        WHERE issue_id IN (SELECT id FROM issue WHERE plan_id = ANY(delete_plan_ids));

        DELETE FROM issue
        WHERE plan_id = ANY(delete_plan_ids);

        -- Delete plan check runs for duplicate plans
        DELETE FROM plan_check_run WHERE plan_id = ANY(delete_plan_ids);

        -- Delete duplicate plans
        DELETE FROM plan WHERE id = ANY(delete_plan_ids);
    END LOOP;
END $$;

-- Drop old non-unique indexes since unique indexes will replace them
DROP INDEX IF EXISTS idx_issue_plan_id;
DROP INDEX IF EXISTS idx_plan_pipeline_id;

-- Add unique constraint on issue.plan_id
CREATE UNIQUE INDEX idx_issue_unique_plan_id ON issue(plan_id);

-- Add unique constraint on plan.pipeline_id
CREATE UNIQUE INDEX idx_plan_unique_pipeline_id ON plan(pipeline_id);
