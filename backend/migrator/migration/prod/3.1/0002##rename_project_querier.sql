UPDATE policy
SET payload = replace(
    payload::text,
    'roles/projectQuerier',
    'roles/sqlEditorUser'
)::jsonb
WHERE type = 'bb.policy.iam';

UPDATE issue
SET payload = replace(
    payload::text,
    'roles/projectQuerier',
    'roles/sqlEditorUser'
)::jsonb
WHERE type = 'bb.issue.grant.request';
