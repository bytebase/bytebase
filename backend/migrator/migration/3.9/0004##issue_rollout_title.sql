-- Step 1: Backfill empty plan names and descriptions with their corresponding issue values
UPDATE plan
SET 
    name = CASE 
        WHEN plan.name = '' THEN issue.name
        ELSE plan.name
    END,
    description = CASE 
        WHEN plan.description = '' THEN issue.description
        ELSE plan.description
    END
FROM issue
WHERE plan.id = issue.plan_id
  AND (plan.name = '' OR plan.description = '');

-- Step 2: Clear issue title (name) and description only when the related plan has non-empty values
UPDATE issue
SET 
    name = CASE 
        WHEN plan.name != '' THEN ''
        ELSE issue.name
    END,
    description = CASE 
        WHEN plan.description != '' THEN ''
        ELSE issue.description
    END
FROM plan
WHERE issue.plan_id = plan.id;

-- Step 3: Clear pipeline title (name) when it's associated with a plan
UPDATE pipeline
SET name = ''
WHERE id IN (
    SELECT DISTINCT pipeline_id 
    FROM plan 
    WHERE pipeline_id IS NOT NULL
);