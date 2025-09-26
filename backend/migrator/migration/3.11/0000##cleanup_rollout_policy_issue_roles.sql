-- Remove deprecated issueRoles field from ROLLOUT policies
-- The issueRoles field contained "roles/CREATOR" and "roles/LAST_APPROVER" which are now deprecated

UPDATE policy
SET payload = payload - 'issueRoles'
WHERE type = 'ROLLOUT'
  AND payload ? 'issueRoles';