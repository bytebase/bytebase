-- 1. Enable requireIssueApproval for ALL projects if ANY rollout policy requires issue approval.
--    This is a "safety first" global backfill: if strictness is used anywhere, enforce it everywhere at the project level
--    to prevent security regression, since environment-level rollout policies are being deprecated/moved.
UPDATE project
SET setting = setting || '{"requireIssueApproval": true}'::jsonb
WHERE EXISTS (
    SELECT 1 FROM policy
    WHERE type = 'ROLLOUT'
    AND (payload::jsonb -> 'checkers' -> 'requiredIssueApproval') = 'true'::jsonb
);

-- 2. Enable requirePlanCheckNoError for ALL projects if ANY rollout policy has planCheckEnforcement configured.
UPDATE project
SET setting = setting || '{"requirePlanCheckNoError": true}'::jsonb
WHERE EXISTS (
    SELECT 1 FROM policy
    WHERE type = 'ROLLOUT'
    AND (payload::jsonb -> 'checkers' -> 'requiredStatusChecks' -> 'planCheckEnforcement') IS NOT NULL
);

-- 3. Cleanup existing checkers and requireIssueApproval from rollout policy in the policy table
UPDATE policy
SET payload = (payload - 'checkers' - 'requireIssueApproval')
WHERE type = 'ROLLOUT';
