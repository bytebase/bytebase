UPDATE policy
SET payload = REPLACE(payload::text, '"users/2"', '"allUsers"')::jsonb
WHERE type = 'bb.policy.project-iam';

DELETE FROM member WHERE principal_id = 2;