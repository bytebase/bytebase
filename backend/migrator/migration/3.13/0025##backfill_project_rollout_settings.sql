-- 1. Enable requireIssueApproval for ALL projects if ANY rollout policy has 'required-issue-approval' config.
--    This is a "safety first" global backfill: if strictness is used anywhere, enforce it everywhere at the project level
--    to prevent security regression, since environment-level rollout policies are being deprecated/moved.
UPDATE project
SET setting = setting || '{"requireIssueApproval": true}'::jsonb
WHERE EXISTS (
    SELECT 1 FROM policy
    WHERE type = 'ROLLOUT'
    AND (payload::jsonb -> 'checkers' -> 'required-issue-approval') IS NOT NULL
);

-- 2. Enable requirePlanCheckNoError for ALL projects if ANY rollout policy has 'plan-check-enforcement' config.
UPDATE project
SET setting = setting || '{"requirePlanCheckNoError": true}'::jsonb
WHERE EXISTS (
    SELECT 1 FROM policy
    WHERE type = 'ROLLOUT'
    AND (payload::jsonb -> 'checkers' -> 'plan-check-enforcement') IS NOT NULL
);

-- 3. Cleanup existing checkers and requireIssueApproval from rollout policy in the policy table
UPDATE policy
SET payload = (payload - 'checkers' - 'requireIssueApproval')
WHERE type = 'ROLLOUT';
